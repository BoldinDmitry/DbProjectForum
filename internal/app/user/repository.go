package user

import "DbProjectForum/internal/app/user/models"

type Repository interface {
	Add(user models.User) error

	GetByNickAndEmail(nickname, email string) ([]models.User, error)
	GetByNick(nickname string) (models.User, error)
	GetUsersByForum(slug string, limit int, since string, desc bool) ([]models.User, error)

	Update(user models.User) (models.User, error)
}
