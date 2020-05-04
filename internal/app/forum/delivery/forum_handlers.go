package delivery

import (
	"DbProjectForum/internal/app/forum"
	"DbProjectForum/internal/app/forum/models"
	"DbProjectForum/internal/pkg/responses"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type forumHandler struct {
	forumRepo forum.Repository
}

func NewForumHandler(r *mux.Router, ur forum.Repository) {
	handler := forumHandler{forumRepo: ur}

	r.HandleFunc("/api/forum/create", handler.Add).Methods("POST")
	r.HandleFunc("/api/forum/{slug}/details", handler.Get).Methods("GET")
	r.HandleFunc("/api/forum/{slug}/create", handler.AddThread).Methods("POST")

	r.HandleFunc("/api/forum/{slug}/threads", handler.GetThreads).Methods("GET")

	r.HandleFunc("/api/thread/{id:[0-9]+}/details", handler.GetThreadDetailsID).Methods("GET")
	r.HandleFunc("/api/thread/{slug}/details", handler.GetThreadDetailsSlug).Methods("GET")

	r.HandleFunc("/api/thread/{slug_or_id}/details", handler.UpdateThreadBySlugOrID).Methods("POST")

	r.HandleFunc("/api/thread/{id:[0-9]+}/create", handler.AddPostID).Methods("POST")
	r.HandleFunc("/api/thread/{slug}/create", handler.AddPostSlug).Methods("POST")
	r.HandleFunc("/api/thread/{slug_or_id}/posts", handler.GetPostsSlug).Methods("GET")

	r.HandleFunc("/api/post/{id:[0-9]+}/details", handler.GetPostByID).Methods("GET")
	r.HandleFunc("/api/post/{id:[0-9]+}/details", handler.UpdatePost).Methods("POST")

	r.HandleFunc("/api/thread/{id:[0-9]+}/vote", handler.AddVoteID).Methods("POST")
	r.HandleFunc("/api/thread/{slug}/vote", handler.AddVoteSlug).Methods("POST")

	r.HandleFunc("/api/service/status", handler.GetServiceStatus).Methods("GET")
	r.HandleFunc("/api/service/clear", handler.ClearDataBase).Methods("POST")
}

func (f *forumHandler) Add(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}
	var newForum models.Forum
	err = json.Unmarshal(data, &newForum)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	newForumDB, err := f.forumRepo.Add(newForum)
	if pgerr, ok := err.(*pq.Error); ok {
		switch pgerr.Code {
		case "23505":
			forumObj, err := f.forumRepo.GetBySlug(newForum.Slug)
			if err != nil {
				responses.SendServerError(err.Error(), w)
				return
			}
			responses.SendResponse(409, forumObj, w)
			return
		case "23503":
			err := responses.HttpError{
				Message: fmt.Sprintf("Can't find user with nickname: %s", newForum.User),
			}
			responses.SendResponse(404, err, w)
			return
		}

	}
	if err == sql.ErrNoRows {
		err := responses.HttpError{
			Message: fmt.Sprintf("Can't find user with nickname: %s", newForum.User),
		}
		responses.SendResponse(404, err, w)
		return
	}

	if err != nil {
		responses.SendResponse(400, err.Error(), w)
		return
	}

	responses.SendResponse(201, newForumDB, w)
}

func (f *forumHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug, found := mux.Vars(r)["slug"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}

	forumObj, err := f.forumRepo.GetBySlug(slug)
	switch err {
	case sql.ErrNoRows:
		err := responses.HttpError{
			Message: fmt.Sprintf("Can't find forum with slug: %s", slug),
		}
		responses.SendResponse(404, err, w)
		return
	case nil:
	default:
		responses.SendServerError(err.Error(), w)
	}

	responses.SendResponseOK(forumObj, w)
	return
}

func (f *forumHandler) AddThread(w http.ResponseWriter, r *http.Request) {
	forumSlug, found := mux.Vars(r)["slug"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}

	newThread := models.Thread{Forum: forumSlug}

	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	err = json.Unmarshal(data, &newThread)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	newThreadDB, err := f.forumRepo.AddThread(newThread)
	if pgerr, ok := err.(*pq.Error); ok && pgerr.Code == "23505" {
		threadOld, err := f.forumRepo.GetThreadBySlug(newThread.Slug.String)
		if err != nil {
			responses.SendServerError(err.Error(), w)
			return
		}
		responses.SendResponse(409, threadOld, w)
		return
	}

	if err != nil {
		errHttp := responses.HttpError{Message: err.Error()}
		responses.SendResponse(404, errHttp, w)
		return
	}

	responses.SendResponse(201, newThreadDB, w)
}

func extractBoolValue(r *http.Request, valueName string) (bool, error) {
	ValueStr, ok := r.URL.Query()[valueName]
	var value bool
	var err error

	if !ok {
		value = false
	} else {
		value, err = strconv.ParseBool(ValueStr[0])
		if err != nil {
			return false, err
		} else if len(ValueStr) > 1 {
			return false, err
		}
	}

	return value, nil
}

func extractIntValue(r *http.Request, valueName string) (int, error) {
	ValueStr, ok := r.URL.Query()[valueName]
	var value int
	var err error

	if !ok {
		value = 0
	} else {
		value, err = strconv.Atoi(ValueStr[0])
		if err != nil {
			return -1, err
		} else if len(ValueStr) > 1 {
			return -1, err
		}
	}

	return value, nil
}

func (f *forumHandler) GetThreads(w http.ResponseWriter, r *http.Request) {
	forumSlug, found := mux.Vars(r)["slug"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}

	limit, err := extractIntValue(r, "limit")
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}
	var since string
	sinceRow, ok := r.URL.Query()["since"]
	if !ok || len(sinceRow) == 0 {
		since = ""
	} else {
		since = sinceRow[0]
	}

	desc, err := extractBoolValue(r, "desc")
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	threads, err := f.forumRepo.GetThreads(forumSlug, limit, since, desc)
	if err == sql.ErrNoRows || len(threads) == 0 {
		exists, err := f.forumRepo.CheckThreadExists(forumSlug)
		if err != nil {
			responses.SendServerError(err.Error(), w)
			return
		}
		if exists {
			data := make([]models.Thread, 0, 0)
			responses.SendResponseOK(data, w)
			return
		}
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf("Can't find forum by slug: %s", forumSlug),
		}
		responses.SendResponse(404, errHTTP, w)
		return
	}

	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}
	responses.SendResponseOK(threads, w)
	return
}

func (f *forumHandler) createPost(w http.ResponseWriter, r *http.Request, id int) {
	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	var newPosts []models.Post
	err = json.Unmarshal(data, &newPosts)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	newPosts, err = f.forumRepo.AddPosts(newPosts, id)

	if err != nil {
		if pgerr, ok := err.(*pq.Error); ok {
			switch pgerr.Code {
			case "00409":
				responses.SendResponse(409, map[int]int{}, w)
				return
			}
		}

		if err == sql.ErrNoRows {
			_, err = f.forumRepo.GetThreadByID(id)
			if err == sql.ErrNoRows {
				responses.SendResponse(404, map[int]int{}, w)
				return
			}
			responses.SendResponse(409, map[int]int{}, w)
			return
		}

		httpError := map[string]string{
			"message": err.Error(),
		}
		responses.SendResponse(404, httpError, w)
		return
	}

	responses.SendResponse(201, newPosts, w)
	return
}

func (f *forumHandler) AddPostSlug(w http.ResponseWriter, r *http.Request) {
	slug, found := mux.Vars(r)["slug"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}
	id, err := f.forumRepo.GetThreadIDBySlug(slug)
	if err != nil {
		if err == sql.ErrNoRows {
			errHTTP := responses.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			responses.SendResponse(404, errHTTP, w)
		}
		responses.SendServerError(err.Error(), w)
		return
	}

	f.createPost(w, r, id)

}

func (f *forumHandler) AddPostID(w http.ResponseWriter, r *http.Request) {
	ValueStr := mux.Vars(r)["id"]

	value, err := strconv.Atoi(ValueStr)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}
	//ToDo maybe can don't do this
	_, err = f.forumRepo.GetThreadByID(value)
	if err != nil {
		if err == sql.ErrNoRows {
			errHTTP := responses.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			responses.SendResponse(404, errHTTP, w)
		}
		responses.SendServerError(err.Error(), w)
		return
	}

	f.createPost(w, r, value)
}

func (f *forumHandler) AddVoteSlug(w http.ResponseWriter, r *http.Request) {
	threadSlug, found := mux.Vars(r)["slug"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	var newVote models.Vote
	err = json.Unmarshal(data, &newVote)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}
	threadID, _ := f.forumRepo.GetThreadIDBySlug(threadSlug)
	newVote.IdThread = int64(threadID)
	err = f.forumRepo.AddVote(newVote)
	if err != nil {
		pgerr, ok := err.(*pq.Error)
		if !ok {
			errHTTP := responses.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			responses.SendResponse(404, errHTTP, w)
			return
		}
		if pgerr.Code != "23505" {
			errHTTP := responses.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			responses.SendResponse(404, errHTTP, w)
			return
		}

	}

	updatedThread, err := f.forumRepo.GetThreadBySlug(threadSlug)
	if err != nil {
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		responses.SendResponse(404, errHTTP, w)
		return
	}

	responses.SendResponseOK(updatedThread, w)
}

func (f *forumHandler) AddVoteID(w http.ResponseWriter, r *http.Request) {
	ValueStr := mux.Vars(r)["id"]

	value, err := strconv.Atoi(ValueStr)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	var newVote models.Vote
	err = json.Unmarshal(data, &newVote)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}
	newVote.IdThread = int64(value)

	err = f.forumRepo.AddVote(newVote)
	if err != nil {
		pgerr, ok := err.(*pq.Error)
		if !ok {
			errHTTP := responses.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			responses.SendResponse(404, errHTTP, w)
			return
		}
		if pgerr.Code != "23505" {
			errHTTP := responses.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			responses.SendResponse(404, errHTTP, w)
			return
		} else {
			err = f.forumRepo.UpdateVote(newVote)
			if err != nil {
				errHTTP := responses.HttpError{
					Message: fmt.Sprintf(err.Error()),
				}
				responses.SendResponse(404, errHTTP, w)
				return
			}
		}
	}
	updatedThread, err := f.forumRepo.GetThreadByID(value)
	if err != nil {
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		responses.SendResponse(404, errHTTP, w)
		return
	}

	responses.SendResponseOK(updatedThread, w)
}

func (f *forumHandler) GetThreadDetailsSlug(w http.ResponseWriter, r *http.Request) {
	threadSlug, found := mux.Vars(r)["slug"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}
	id, err := f.forumRepo.GetThreadIDBySlug(threadSlug)
	if err != nil {
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		responses.SendResponse(404, errHTTP, w)
		return
	}
	forumObj, err := f.forumRepo.GetThreadByID(id)
	if err != nil {
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		responses.SendResponse(404, errHTTP, w)
		return
	}

	responses.SendResponseOK(forumObj, w)
	return
}

func (f *forumHandler) GetThreadDetailsID(w http.ResponseWriter, r *http.Request) {
	ValueStr := mux.Vars(r)["id"]

	id, err := strconv.Atoi(ValueStr)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}
	forumObj, err := f.forumRepo.GetThreadByID(id)
	if err != nil {
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		responses.SendResponse(404, errHTTP, w)
		return
	}

	responses.SendResponseOK(forumObj, w)
	return
}

func (f *forumHandler) UpdateThreadBySlugOrID(w http.ResponseWriter, r *http.Request) {
	threadSlugOrID, found := mux.Vars(r)["slug_or_id"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}
	var newThread models.Thread
	if id, err := strconv.Atoi(threadSlugOrID); err == nil {
		newThread.Id = int32(id)
	} else {
		newThread.Slug = models.JsonNullString{
			NullString: sql.NullString{Valid: true, String: threadSlugOrID},
		}
	}

	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	err = json.Unmarshal(data, &newThread)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	thread, err := f.forumRepo.UpdateThread(newThread)
	if err != nil {
		responses.SendResponse(404, err, w)
		return
	}

	responses.SendResponseOK(thread, w)
	return
}

func (f *forumHandler) GetPostsSlug(w http.ResponseWriter, r *http.Request) {
	threadSlugOrID, found := mux.Vars(r)["slug_or_id"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}

	limit, err := extractIntValue(r, "limit")
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	since, err := extractIntValue(r, "since")
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	var sortType string
	if len(r.URL.Query()["sort"]) != 0 {
		sortType = r.URL.Query()["sort"][0]
	} else {
		sortType = "flat"
	}

	desc, err := extractBoolValue(r, "desc")
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}
	var slugOrID models.Thread
	if id, err := strconv.Atoi(threadSlugOrID); err == nil {
		slugOrID.Id = int32(id)
	}
	slug := sql.NullString{String: threadSlugOrID, Valid: true}
	slugJSON := models.JsonNullString{NullString: slug}
	slugOrID.Slug = slugJSON

	posts, err := f.forumRepo.GetPosts(slugOrID, limit, since, sortType, desc)
	if err != nil {
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		responses.SendResponse(404, errHTTP, w)
		return
	}

	if posts == nil {
		if slugOrID.Id != 0 {
			_, err := f.forumRepo.GetThreadByID(int(slugOrID.Id))
			if err == sql.ErrNoRows {
				httpErr := responses.HttpError{Message: err.Error()}
				responses.SendResponse(404, httpErr, w)
				return
			}
		}
		responses.SendResponseOK([]int{}, w)
		return
	}

	responses.SendResponseOK(posts, w)
	return
}

func (f *forumHandler) GetPostByID(w http.ResponseWriter, r *http.Request) {
	ValueStr := mux.Vars(r)["id"]

	id, err := strconv.Atoi(ValueStr)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	var related string
	if len(r.URL.Query()["related"]) != 0 {
		related = r.URL.Query()["related"][0]
	}

	post, err := f.forumRepo.GetPost(id, strings.Split(related, ","))
	if err != nil {
		httpErr := responses.HttpError{Message: err.Error()}
		responses.SendResponse(404, httpErr, w)
		return
	}

	responses.SendResponseOK(post, w)
	return
}

func (f *forumHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	ValueStr := mux.Vars(r)["id"]

	id, err := strconv.Atoi(ValueStr)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	newPost := models.Post{
		Id: int64(id),
	}

	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	err = json.Unmarshal(data, &newPost)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	newPost, err = f.forumRepo.UpdatePost(newPost)
	if err != nil {
		httpErr := responses.HttpError{Message: err.Error()}
		responses.SendResponse(404, httpErr, w)
		return
	}

	responses.SendResponseOK(newPost, w)
	return
}

func (f *forumHandler) GetServiceStatus(w http.ResponseWriter, r *http.Request) {
	info, err := f.forumRepo.GetServiceStatus()
	if err != nil {
		responses.SendResponse(404, err.Error(), w)
		return
	}
	responses.SendResponseOK(info, w)
	return
}

func (f *forumHandler) ClearDataBase(w http.ResponseWriter, r *http.Request) {
	err := f.forumRepo.ClearDatabase()
	if err != nil {
		responses.SendResponse(404, err.Error(), w)
		return
	}
	responses.SendResponseOK("", w)
	return
}
