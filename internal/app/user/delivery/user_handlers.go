package delivery

import (
	"DbProjectForum/internal/app/forum"
	"DbProjectForum/internal/app/user"
	"DbProjectForum/internal/app/user/models"
	"DbProjectForum/internal/pkg/responses"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"io/ioutil"
	"net/http"
	"strconv"
)

type userHandler struct {
	userRepo  user.Repository
	forumRepo forum.Repository
}

func NewUserHandler(r *mux.Router, ur user.Repository, fr forum.Repository) {
	handler := userHandler{
		userRepo:  ur,
		forumRepo: fr,
	}

	r.HandleFunc("/api/user/{nickname}/create", handler.Add).Methods("POST")
	r.HandleFunc("/api/user/{nickname}/profile", handler.Get).Methods("GET")
	r.HandleFunc("/api/user/{nickname}/profile", handler.Update).Methods("POST")

	r.HandleFunc("/api/forum/{slug}/users", handler.GetByForum).Methods("GET")
}

func (ur *userHandler) Add(w http.ResponseWriter, r *http.Request) {
	nickname, found := mux.Vars(r)["nickname"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}

	newUser := models.User{Nickname: nickname}

	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	err = json.Unmarshal(data, &newUser)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	err = ur.userRepo.Add(newUser)
	if pgerr, ok := err.(*pq.Error); ok && pgerr.Code == "23505" {
		users, err := ur.userRepo.GetByNickAndEmail(newUser.Nickname, newUser.Email)
		if err != nil {
			responses.SendServerError(err.Error(), w)
		}
		responses.SendResponse(409, users, w)
		return
	}

	if err != nil {
		responses.SendResponse(400, err.Error(), w)
		return
	}

	responses.SendResponse(201, newUser, w)
	return
}

func (ur *userHandler) Get(w http.ResponseWriter, r *http.Request) {
	nickname, found := mux.Vars(r)["nickname"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}

	userObj, err := ur.userRepo.GetByNick(nickname)
	if err != nil {
		if err == sql.ErrNoRows {
			err := responses.HttpError{
				Message: fmt.Sprintf("Can't find user by nickname: %s", nickname),
			}
			responses.SendResponse(404, err, w)
			return
		}
		responses.SendServerError(err.Error(), w)
		return
	}

	responses.SendResponseOK(userObj, w)
	return
}

func (ur *userHandler) Update(w http.ResponseWriter, r *http.Request) {
	nickname, found := mux.Vars(r)["nickname"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}

	newUser := models.User{Nickname: nickname}

	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	err = json.Unmarshal(data, &newUser)
	if err != nil {
		responses.SendServerError(err.Error(), w)
		return
	}

	userDB, err := ur.userRepo.Update(newUser)
	if pgerr, ok := err.(*pq.Error); ok {
		switch pgerr.Code {
		case "23505":
			err := responses.HttpError{
				Message: fmt.Sprintf("This email is already registered by user: %s", newUser.Email),
			}
			responses.SendResponse(409, err, w)
			return
		}
	}
	if err != nil {
		if err == sql.ErrNoRows {
			err := responses.HttpError{
				Message: fmt.Sprintf("Can't find user by nickname: %s", newUser.Nickname),
			}
			responses.SendResponse(404, err, w)
			return
		}
		responses.SendServerError(err.Error(), w)
	}

	responses.SendResponseOK(userDB, w)
	return
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

func (ur *userHandler) GetByForum(w http.ResponseWriter, r *http.Request) {
	slug, found := mux.Vars(r)["slug"]
	if !found {
		responses.SendResponse(400, "bad request", w)
		return
	}

	limit, err := extractIntValue(r, "limit")
	if err != nil {
		responses.SendResponse(400, err, w)
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
		responses.SendResponse(400, err, w)
		return
	}

	users, err := ur.userRepo.GetUsersByForum(slug, limit, since, desc)
	if err != nil {
		responses.SendResponse(404, err, w)
		return
	}

	if users == nil {
		_, err = ur.forumRepo.GetBySlug(slug)
		if err != nil {
			responses.SendResponse(404, err, w)
			return
		}
		responses.SendResponseOK([]models.User{}, w)
		return
	}

	responses.SendResponseOK(users, w)
	return
}
