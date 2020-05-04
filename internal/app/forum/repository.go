package forum

import (
	"DbProjectForum/internal/app/forum/models"
)

type Repository interface {
	Add(forum models.Forum) (models.Forum, error)
	GetBySlug(slug string) (models.Forum, error)

	AddThread(thread models.Thread) (models.Thread, error)
	UpdateThread(newThread models.Thread) (models.Thread, error)
	GetThreads(slug string, limit int, since string, desc bool) ([]models.Thread, error)
	CheckThreadExists(slug string) (bool, error)
	GetThreadBySlug(slug string) (models.Thread, error)
	GetThreadByID(id int) (models.Thread, error)
	GetThreadIDBySlug(slug string) (int, error)
	GetThreadSlugByID(id int) (string, error)

	AddPosts(posts []models.Post, threadID int) ([]models.Post, error)
	GetPosts(postSlugOrId models.Thread, limit, since int, sort string, desc bool) ([]models.Post, error)
	GetPost(id int, related []string) (map[string]interface{}, error)
	UpdatePost(newPost models.Post) (models.Post, error)

	AddVote(vote models.Vote) error
	UpdateVote(vote models.Vote) error

	GetServiceStatus() (map[string]int, error)
	ClearDatabase() error
}
