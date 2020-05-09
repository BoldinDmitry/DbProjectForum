package delivery

import (
	"DbProjectForum/internal/app/forum"
	"DbProjectForum/internal/app/user"
	"DbProjectForum/internal/app/user/models"
	"DbProjectForum/internal/pkg/responses"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/fasthttp/router"
	"github.com/lib/pq"
	"github.com/valyala/fasthttp"
	"strconv"
)

type userHandler struct {
	userRepo  user.Repository
	forumRepo forum.Repository
}

func NewUserHandler(r *router.Router, ur user.Repository, fr forum.Repository) {
	handler := userHandler{
		userRepo:  ur,
		forumRepo: fr,
	}

	r.POST("/api/user/{nickname}/create", handler.Add)
	r.GET("/api/user/{nickname}/profile", handler.Get)
	r.POST("/api/user/{nickname}/profile", handler.Update)

	r.GET("/api/forum/{slug}/users", handler.GetByForum)
}

func (ur *userHandler) Add(ctx *fasthttp.RequestCtx) {
	nickname, ok := ctx.UserValue("nickname").(string)
	if !ok {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	newUser := models.User{Nickname: nickname}

	err := json.Unmarshal(ctx.PostBody(), &newUser)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	err = ur.userRepo.Add(newUser)
	if pgerr, ok := err.(*pq.Error); ok && pgerr.Code == "23505" {
		users, err := ur.userRepo.GetByNickAndEmail(newUser.Nickname, newUser.Email)
		if err != nil {
			responses.SendServerError(err.Error(), ctx)
		}
		responses.SendResponse(409, users, ctx)
		return
	}

	if err != nil {
		responses.SendResponse(400, err.Error(), ctx)
		return
	}

	responses.SendResponse(201, newUser, ctx)
	return
}

func (ur *userHandler) Get(ctx *fasthttp.RequestCtx) {
	nickname, found := ctx.UserValue("nickname").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	userObj, err := ur.userRepo.GetByNick(nickname)
	if err != nil {
		if err == sql.ErrNoRows {
			err := responses.HttpError{
				Message: fmt.Sprintf("Can't find user by nickname: %s", nickname),
			}
			responses.SendResponse(404, err, ctx)
			return
		}
		responses.SendServerError(err.Error(), ctx)
		return
	}

	responses.SendResponseOK(userObj, ctx)
	return
}

func (ur *userHandler) Update(ctx *fasthttp.RequestCtx) {
	nickname, found := ctx.UserValue("nickname").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	newUser := models.User{Nickname: nickname}

	err := json.Unmarshal(ctx.PostBody(), &newUser)
	if err != nil {
		responses.SendServerError(err.Error(), ctx)
		return
	}

	userDB, err := ur.userRepo.Update(newUser)
	if pgerr, ok := err.(*pq.Error); ok {
		switch pgerr.Code {
		case "23505":
			err := responses.HttpError{
				Message: fmt.Sprintf("This email is already registered by user: %s", newUser.Email),
			}
			responses.SendResponse(409, err, ctx)
			return
		}
	}
	if err != nil {
		if err == sql.ErrNoRows {
			err := responses.HttpError{
				Message: fmt.Sprintf("Can't find user by nickname: %s", newUser.Nickname),
			}
			responses.SendResponse(404, err, ctx)
			return
		}
		responses.SendServerError(err.Error(), ctx)
	}

	responses.SendResponseOK(userDB, ctx)
	return
}

func extractBoolValue(ctx *fasthttp.RequestCtx, valueName string) (bool, error) {
	ValueStr := string(ctx.QueryArgs().Peek(valueName))
	var value bool
	var err error

	value, err = strconv.ParseBool(ValueStr)
	if err != nil {
		return false, err
	} else if len(ValueStr) > 1 {
		return false, err
	}

	return value, nil
}

func extractIntValue(ctx *fasthttp.RequestCtx, valueName string) (int, error) {
	ValueStr := string(ctx.QueryArgs().Peek(valueName))
	var value int
	var err error

	value, err = strconv.Atoi(ValueStr)
	if err != nil {
		return -1, err
	} else if len(ValueStr) > 1 {
		return -1, err
	}

	return value, nil
}

func (ur *userHandler) GetByForum(ctx *fasthttp.RequestCtx) {
	slug, found := ctx.UserValue("slug").(string)
	if !found {
		responses.SendResponse(400, "bad request", ctx)
		return
	}

	limit, err := extractIntValue(ctx, "limit")
	if err != nil {
		responses.SendResponse(400, err, ctx)
		return
	}

	since := string(ctx.QueryArgs().Peek("since"))

	desc, err := extractBoolValue(ctx, "desc")
	if err != nil {
		responses.SendResponse(400, err, ctx)
		return
	}

	users, err := ur.userRepo.GetUsersByForum(slug, limit, since, desc)
	if err != nil {
		responses.SendResponse(404, err, ctx)
		return
	}

	if users == nil {
		_, err = ur.forumRepo.GetBySlug(slug)
		if err != nil {
			responses.SendResponse(404, err, ctx)
			return
		}
		responses.SendResponseOK([]models.User{}, ctx)
		return
	}

	responses.SendResponseOK(users, ctx)
	return
}
