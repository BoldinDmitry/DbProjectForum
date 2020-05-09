package repository

import (
	"DbProjectForum/internal/app/user"
	"DbProjectForum/internal/app/user/models"
	"fmt"
	"github.com/jmoiron/sqlx"
)

type postgresUserRepository struct {
	Conn *sqlx.DB
}

func NewPostgresCafeRepository(conn *sqlx.DB) user.Repository {
	return &postgresUserRepository{
		Conn: conn,
	}
}

func (p *postgresUserRepository) Add(user models.User) error {
	query := `INSERT INTO users(
    about,
    email,
    fullname,
    nickname)
	VALUES ($1, $2, $3, $4)`

	_, err := p.Conn.Exec(query, user.About, user.Email, user.FullName, user.Nickname)
	return err
}

func (p *postgresUserRepository) GetByNickAndEmail(nickname, email string) ([]models.User, error) {
	query := `SELECT * FROM users WHERE LOWER(Nickname)=LOWER($1) OR Email=$2`

	var data []models.User
	err := p.Conn.Select(&data, query, nickname, email)
	return data, err
}

func (p *postgresUserRepository) GetByNick(nickname string) (models.User, error) {
	query := `SELECT * FROM users WHERE LOWER(Nickname)=LOWER($1)`

	var userObj models.User
	err := p.Conn.Get(&userObj, query, nickname)
	return userObj, err
}

func (p *postgresUserRepository) Update(user models.User) (models.User, error) {
	query := `UPDATE users SET 
                 about=COALESCE(NULLIF($1, ''), about),
                 email=COALESCE(NULLIF($2, ''), email),
                 fullname=COALESCE(NULLIF($3, ''), fullname) 
	WHERE LOWER(nickname) = LOWER($4) RETURNING *`

	var userObj models.User
	err := p.Conn.Get(&userObj, query, user.About, user.Email, user.FullName, user.Nickname)
	return userObj, err
}

func (p *postgresUserRepository) GetUsersByForum(slug string, limit int, since string, desc bool) ([]models.User, error) {
	var query string
	if desc {
		if since != "" {
			query = fmt.Sprintf(`SELECT users.about, users.Email, users.FullName, users.Nickname FROM users
    	inner join users_forum uf on users.Nickname = uf.nickname
        WHERE uf.slug =$1 AND uf.nickname < '%s'
        ORDER BY lower(users.Nickname) DESC LIMIT NULLIF($2, 0)`, since)
		} else {
			query = `SELECT users.about, users.Email, users.FullName, users.Nickname FROM users
    	inner join users_forum uf on users.Nickname = uf.nickname
        WHERE uf.slug =$1
        ORDER BY lower(users.Nickname) DESC LIMIT NULLIF($2, 0)`
		}
	} else {
		query = fmt.Sprintf(`SELECT users.about, users.Email, users.FullName, users.Nickname FROM users
    	inner join users_forum uf on users.Nickname = uf.nickname
        WHERE uf.slug =$1 AND uf.nickname > '%s'
        ORDER BY lower(users.Nickname) LIMIT NULLIF($2, 0)`, since)
	}
	var data []models.User
	err := p.Conn.Select(&data, query, slug, limit)
	return data, err
}
