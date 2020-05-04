package main

import (
	"DbProjectForum/configs"
	_forumHandlers "DbProjectForum/internal/app/forum/delivery"
	_forumRepo "DbProjectForum/internal/app/forum/repository"
	_userHandlers "DbProjectForum/internal/app/user/delivery"
	_userRepo "DbProjectForum/internal/app/user/repository"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

func applicationJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func main() {
	r := mux.NewRouter()

	r.Use(applicationJSON)

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

	http.Handle("/", r)
	log.Info().Msgf("starting server at :5000")
	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:5000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Error().Msgf(srv.ListenAndServe().Error())
}
