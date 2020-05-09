package delivery

import (
	"DbProjectForum/internal/app/forum"
	"DbProjectForum/internal/app/forum/models"
	"DbProjectForum/internal/pkg/responses"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/fasthttp/router"
	"github.com/lib/pq"
	"github.com/valyala/fasthttp"
	"strconv"
	"strings"
)

type forumHandler struct {
	forumRepo forum.Repository
}

func NewForumHandler(r *router.Router, ur forum.Repository) {
	handler := forumHandler{forumRepo: ur}

	r.POST("/api/forum/create", handler.Add)
	r.GET("/api/forum/{slug}/details", handler.Get)
	r.POST("/api/forum/{slug}/create", handler.AddThread)

	r.GET("/api/forum/{slug}/threads", handler.GetThreads)

	r.GET("/api/thread/{id:[0-9]+}/details", handler.GetThreadDetailsID)
	r.GET("/api/thread/{slug}/details", handler.GetThreadDetailsSlug)

	r.POST("/api/thread/{slug_or_id}/details", handler.UpdateThreadBySlugOrID)

	r.POST("/api/thread/{slug_or_id}/create", handler.AddPostSlug)
	r.GET("/api/thread/{slug_or_id}/posts", handler.GetPostsSlug)

	r.GET("/api/post/{id:[0-9]+}/details", handler.GetPostByID)
	r.POST("/api/post/{id:[0-9]+}/details", handler.UpdatePost)

	r.POST("/api/thread/{id:[0-9]+}/vote", handler.AddVoteID)
	r.POST("/api/thread/{slug}/vote", handler.AddVoteSlug)

	r.GET("/api/service/status", handler.GetServiceStatus)
	r.POST("/api/service/clear", handler.ClearDataBase)
}

func (f *forumHandler) Add(ctx *fasthttp.RequestCtx) {
	var newForum models.Forum
	err := json.Unmarshal(ctx.PostBody(), &newForum)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	newForumDB, err := f.forumRepo.Add(newForum)
	if pgerr, ok := err.(*pq.Error); ok {
		switch pgerr.Code {
		case "23505":
			forumObj, err := f.forumRepo.GetBySlug(newForum.Slug)
			if err != nil {
				responses.SendServerError(err.Error(), ctx)
				return
			}
			responses.SendResponse(409, forumObj, ctx)
			return
		case "23503":
			err := responses.HttpError{
				Message: fmt.Sprintf("Can't find user with nickname: %s", newForum.User),
			}
			responses.SendResponse(404, err, ctx)
			return
		}

	}
	if err == sql.ErrNoRows {
		err := responses.HttpError{
			Message: fmt.Sprintf("Can't find user with nickname: %s", newForum.User),
		}
		responses.SendResponse(404, err, ctx)
		return
	}

	if err != nil {
		responses.SendResponse(400, err.Error(), ctx)
		return
	}

	responses.SendResponse(201, newForumDB, ctx)
}

func (f *forumHandler) Get(ctx *fasthttp.RequestCtx) {
	slug, ok := ctx.UserValue("slug").(string)
	if !ok {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	forumObj, err := f.forumRepo.GetBySlug(slug)
	switch err {
	case sql.ErrNoRows:
		err := responses.HttpError{
			Message: fmt.Sprintf("Can't find forum with slug: %s", slug),
		}
		responses.SendResponse(404, err, ctx)
		return
	case nil:
	default:
		responses.SendServerError(err.Error(), ctx)
	}

	responses.SendResponseOK(forumObj, ctx)
	return
}

func (f *forumHandler) AddThread(ctx *fasthttp.RequestCtx) {
	forumSlug, found := ctx.UserValue("slug").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	newThread := models.Thread{Forum: forumSlug}

	err := json.Unmarshal(ctx.PostBody(), &newThread)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	newThreadDB, err := f.forumRepo.AddThread(newThread)
	if pgerr, ok := err.(*pq.Error); ok && pgerr.Code == "23505" {
		threadOld, err := f.forumRepo.GetThreadBySlug(newThread.Slug.String)
		if err != nil {
			responses.SendServerError(err.Error(), ctx)
			return
		}
		responses.SendResponse(409, threadOld, ctx)
		return
	}

	if err != nil {
		errHttp := responses.HttpError{Message: err.Error()}
		responses.SendResponse(404, errHttp, ctx)
		return
	}

	responses.SendResponse(201, newThreadDB, ctx)
}

func extractBoolValue(ctx *fasthttp.RequestCtx, valueName string) (bool, error) {
	ValueStr := string(ctx.QueryArgs().Peek(valueName))
	var value bool
	var err error

	if ValueStr == "" {
		return false, nil
	}
	value, err = strconv.ParseBool(ValueStr)
	if err != nil {
		return false, err
	}

	return value, nil
}

func extractIntValue(ctx *fasthttp.RequestCtx, valueName string) (int, error) {
	ValueStr := string(ctx.QueryArgs().Peek(valueName))
	var value int
	var err error

	if ValueStr == "" {
		return 0, nil
	}

	value, err = strconv.Atoi(ValueStr)
	if err != nil {
		return -1, err
	}

	return value, nil
}

func (f *forumHandler) GetThreads(ctx *fasthttp.RequestCtx) {
	forumSlug, found := ctx.UserValue("slug").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	limit, err := extractIntValue(ctx, "limit")
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	since := string(ctx.QueryArgs().Peek("since"))

	desc, err := extractBoolValue(ctx, "desc")
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}
	fmt.Println("ALL:", limit, since, desc, forumSlug)
	threads, err := f.forumRepo.GetThreads(forumSlug, limit, since, desc)
	if err == sql.ErrNoRows || len(threads) == 0 {
		exists, err := f.forumRepo.CheckThreadExists(forumSlug)
		if err != nil {
			responses.SendServerError(err.Error(), ctx)
			return
		}
		if exists {
			data := make([]models.Thread, 0, 0)
			responses.SendResponseOK(data, ctx)
			return
		}
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf("Can't find forum by slug: %s", forumSlug),
		}
		responses.SendResponse(404, errHTTP, ctx)
		return
	}

	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}
	responses.SendResponseOK(threads, ctx)
	return
}

func (f *forumHandler) createPost(ctx *fasthttp.RequestCtx, id int) {
	var newPosts []models.Post
	err := json.Unmarshal(ctx.PostBody(), &newPosts)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	newPosts, err = f.forumRepo.AddPosts(newPosts, id)

	if err != nil {
		if pgerr, ok := err.(*pq.Error); ok {
			switch pgerr.Code {
			case "00409":
				responses.SendResponse(409, map[int]int{}, ctx)
				return
			}
		}

		if err == sql.ErrNoRows {
			_, err = f.forumRepo.GetThreadByID(id)
			if err == sql.ErrNoRows {
				responses.SendResponse(404, map[int]int{}, ctx)
				return
			}
			responses.SendResponse(409, map[int]int{}, ctx)
			return
		}

		httpError := map[string]string{
			"message": err.Error(),
		}
		responses.SendResponse(404, httpError, ctx)
		return
	}

	responses.SendResponse(201, newPosts, ctx)
	return
}

func (f *forumHandler) AddPostSlug(ctx *fasthttp.RequestCtx) {
	slugOrId, found := ctx.UserValue("slug_or_id").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}
	var id int
	id, err := strconv.Atoi(slugOrId)
	if err == nil {
		_, err = f.forumRepo.GetThreadByID(id)
		if err != nil {
			errHTTP := responses.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			responses.SendResponse(404, errHTTP, ctx)

			return
		}
	} else {
		id, err = f.forumRepo.GetThreadIDBySlug(slugOrId)
		if err != nil {
			errHTTP := responses.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			responses.SendResponse(404, errHTTP, ctx)
			return
		}
	}

	f.createPost(ctx, id)

}

func (f *forumHandler) AddVoteSlug(ctx *fasthttp.RequestCtx) {
	threadSlug, found := ctx.UserValue("slug").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	var newVote models.Vote
	err := json.Unmarshal(ctx.PostBody(), &newVote)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
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
			responses.SendResponse(404, errHTTP, ctx)
			return
		}
		if pgerr.Code != "23505" {
			errHTTP := responses.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			responses.SendResponse(404, errHTTP, ctx)
			return
		}

	}

	updatedThread, err := f.forumRepo.GetThreadBySlug(threadSlug)
	if err != nil {
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		responses.SendResponse(404, errHTTP, ctx)
		return
	}

	responses.SendResponseOK(updatedThread, ctx)
}

func (f *forumHandler) AddVoteID(ctx *fasthttp.RequestCtx) {
	ValueStr, found := ctx.UserValue("id").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	value, err := strconv.Atoi(ValueStr)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	var newVote models.Vote
	err = json.Unmarshal(ctx.PostBody(), &newVote)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
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
			responses.SendResponse(404, errHTTP, ctx)
			return
		}
		if pgerr.Code != "23505" {
			errHTTP := responses.HttpError{
				Message: fmt.Sprintf(err.Error()),
			}
			responses.SendResponse(404, errHTTP, ctx)
			return
		} else {
			err = f.forumRepo.UpdateVote(newVote)
			if err != nil {
				errHTTP := responses.HttpError{
					Message: fmt.Sprintf(err.Error()),
				}
				responses.SendResponse(404, errHTTP, ctx)
				return
			}
		}
	}
	updatedThread, err := f.forumRepo.GetThreadByID(value)
	if err != nil {
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		responses.SendResponse(404, errHTTP, ctx)
		return
	}

	responses.SendResponseOK(updatedThread, ctx)
}

func (f *forumHandler) GetThreadDetailsSlug(ctx *fasthttp.RequestCtx) {
	threadSlug, found := ctx.UserValue("slug").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}
	id, err := f.forumRepo.GetThreadIDBySlug(threadSlug)
	if err != nil {
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		responses.SendResponse(404, errHTTP, ctx)
		return
	}
	forumObj, err := f.forumRepo.GetThreadByID(id)
	if err != nil {
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		responses.SendResponse(404, errHTTP, ctx)
		return
	}

	responses.SendResponseOK(forumObj, ctx)
	return
}

func (f *forumHandler) GetThreadDetailsID(ctx *fasthttp.RequestCtx) {
	ValueStr, found := ctx.UserValue("id").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	id, err := strconv.Atoi(ValueStr)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}
	forumObj, err := f.forumRepo.GetThreadByID(id)
	if err != nil {
		errHTTP := responses.HttpError{
			Message: fmt.Sprintf(err.Error()),
		}
		responses.SendResponse(404, errHTTP, ctx)
		return
	}

	responses.SendResponseOK(forumObj, ctx)
	return
}

func (f *forumHandler) UpdateThreadBySlugOrID(ctx *fasthttp.RequestCtx) {
	threadSlugOrID, found := ctx.UserValue("slug_or_id").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
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

	err := json.Unmarshal(ctx.PostBody(), &newThread)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	thread, err := f.forumRepo.UpdateThread(newThread)
	if err != nil {
		responses.SendResponse(404, err, ctx)
		return
	}

	responses.SendResponseOK(thread, ctx)
	return
}

func (f *forumHandler) GetPostsSlug(ctx *fasthttp.RequestCtx) {
	threadSlugOrID, found := ctx.UserValue("slug_or_id").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	limit, err := extractIntValue(ctx, "limit")
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	since, err := extractIntValue(ctx, "since")
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	sortType := string(ctx.QueryArgs().Peek("sort"))
	if sortType == "" {
		sortType = "flat"
	}

	desc, err := extractBoolValue(ctx, "desc")
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
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
		responses.SendResponse(404, errHTTP, ctx)
		return
	}

	if posts == nil {
		if slugOrID.Id != 0 {
			_, err := f.forumRepo.GetThreadByID(int(slugOrID.Id))
			if err == sql.ErrNoRows {
				httpErr := responses.HttpError{Message: err.Error()}
				responses.SendResponse(404, httpErr, ctx)
				return
			}
		}
		responses.SendResponseOK([]int{}, ctx)
		return
	}

	responses.SendResponseOK(posts, ctx)
	return
}

func (f *forumHandler) GetPostByID(ctx *fasthttp.RequestCtx) {
	ValueStr, found := ctx.UserValue("id").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	id, err := strconv.Atoi(ValueStr)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	related := string(ctx.QueryArgs().Peek("related"))

	post, err := f.forumRepo.GetPost(id, strings.Split(related, ","))
	if err != nil {
		httpErr := responses.HttpError{Message: err.Error()}
		responses.SendResponse(404, httpErr, ctx)
		return
	}

	responses.SendResponseOK(post, ctx)
	return
}

func (f *forumHandler) UpdatePost(ctx *fasthttp.RequestCtx) {
	ValueStr, found := ctx.UserValue("id").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	id, err := strconv.Atoi(ValueStr)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	newPost := models.Post{
		Id: int64(id),
	}

	err = json.Unmarshal(ctx.PostBody(), &newPost)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	newPost, err = f.forumRepo.UpdatePost(newPost)
	if err != nil {
		httpErr := responses.HttpError{Message: err.Error()}
		responses.SendResponse(404, httpErr, ctx)
		return
	}

	responses.SendResponseOK(newPost, ctx)
	return
}

func (f *forumHandler) GetServiceStatus(ctx *fasthttp.RequestCtx) {
	info, err := f.forumRepo.GetServiceStatus()
	if err != nil {
		responses.SendResponse(404, err.Error(), ctx)
		return
	}
	responses.SendResponseOK(info, ctx)
	return
}

func (f *forumHandler) ClearDataBase(ctx *fasthttp.RequestCtx) {
	err := f.forumRepo.ClearDatabase()
	if err != nil {
		responses.SendResponse(404, err.Error(), ctx)
		return
	}
	responses.SendResponseOK("", ctx)
	return
}
