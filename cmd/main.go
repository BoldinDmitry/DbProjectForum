package main

import (
	"DbProjectForum/configs"
	_forumHandlers "DbProjectForum/internal/app/forum/delivery"
	_forumRepo "DbProjectForum/internal/app/forum/repository"
	_userHandlers "DbProjectForum/internal/app/user/delivery"
	_userRepo "DbProjectForum/internal/app/user/repository"
	"fmt"
	"github.com/fasthttp/router"
	"github.com/jackc/pgx"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

func applicationJSON(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Content-Type", "application/json")
		next(ctx)
	}
}

//func loggingMiddleware(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		rec := responses.StatusRecorder{ResponseWriter: w, Status: 200}
//
//		next.ServeHTTP(&rec, r)
//		msg := fmt.Sprintf("URL: %s, METHOD: %s, Answer code: %d", r.RequestURI, r.Method, rec.Status)
//		log.Info().Msgf(msg)
//	})
//}

func main() {
	r := router.New()

	//r.Use(applicationJSON)
	//r.Use(loggingMiddleware)

	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable port=%s",
		configs.PostgresPreferences.User,
		configs.PostgresPreferences.Password,
		configs.PostgresPreferences.DBName,
		configs.PostgresPreferences.Port)

	pgxConn, err := pgx.ParseConnectionString(connStr)
	if err != nil {
		log.Error().Msgf(err.Error())
		return
	}

	pgxConn.PreferSimpleProtocol = true

	config := pgx.ConnPoolConfig{
		ConnConfig:     pgxConn,
		MaxConnections: 100,
		AfterConnect:   nil,
		AcquireTimeout: 0,
	}

	connPool, err := pgx.NewConnPool(config)
	if err != nil {
		log.Error().Msgf(err.Error())
	}

	userRepo := _userRepo.NewPostgresCafeRepository(connPool)
	forumRepo := _forumRepo.NewPostgresForumRepository(connPool, userRepo)

	_userHandlers.NewUserHandler(r, userRepo, forumRepo)
	_forumHandlers.NewForumHandler(r, forumRepo, userRepo)

	log.Error().Msgf(fasthttp.ListenAndServe(":5000", applicationJSON(r.Handler)).Error())
}
