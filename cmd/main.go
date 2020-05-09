package main

import (
	"DbProjectForum/configs"
	_forumHandlers "DbProjectForum/internal/app/forum/delivery"
	_forumRepo "DbProjectForum/internal/app/forum/repository"
	_userHandlers "DbProjectForum/internal/app/user/delivery"
	_userRepo "DbProjectForum/internal/app/user/repository"
	"fmt"
	"github.com/fasthttp/router"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

func applicationJSON(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		fmt.Println(ctx.URI())
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

	connStr := fmt.Sprintf("user=%s password=%s dbname=postgres sslmode=disable port=%s",
		configs.PostgresPreferences.User,
		configs.PostgresPreferences.Password,
		configs.PostgresPreferences.Port)

	conn, err := sqlx.Open("postgres", connStr)
	if err != nil {
		log.Error().Msgf(err.Error())
	}

	userRepo := _userRepo.NewPostgresCafeRepository(conn)
	forumRepo := _forumRepo.NewPostgresForumRepository(conn, userRepo)

	_userHandlers.NewUserHandler(r, userRepo, forumRepo)
	_forumHandlers.NewForumHandler(r, forumRepo)

	log.Error().Msgf(fasthttp.ListenAndServe(":5000", applicationJSON(r.Handler)).Error())
}
