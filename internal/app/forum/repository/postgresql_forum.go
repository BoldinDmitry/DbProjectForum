package repository

import (
	"DbProjectForum/internal/app/forum"
	"DbProjectForum/internal/app/forum/models"
	"DbProjectForum/internal/app/user"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
)

type postgresForumRepository struct {
	conn     *sqlx.DB
	userRepo user.Repository
}

func NewPostgresForumRepository(conn *sqlx.DB, repository user.Repository) forum.Repository {
	return &postgresForumRepository{
		conn:     conn,
		userRepo: repository,
	}
}

func (p *postgresForumRepository) Add(forum models.Forum) (models.Forum, error) {
	query := `INSERT INTO forum(
    "user",
    slug,
    title)
	VALUES ($1, $2, $3) RETURNING *`

	userObj, err := p.userRepo.GetByNick(forum.User)
	if err != nil {
		return models.Forum{}, err
	}

	var forumObj models.Forum
	err = p.conn.Get(&forumObj, query, userObj.Nickname, forum.Slug, forum.Title)
	return forumObj, err
}

func (p *postgresForumRepository) GetBySlug(slug string) (models.Forum, error) {
	query := `SELECT * FROM forum WHERE LOWER(slug)=LOWER($1)`

	var forumObj models.Forum
	err := p.conn.Get(&forumObj, query, slug)

	return forumObj, err
}

func (p *postgresForumRepository) AddThread(thread models.Thread) (models.Thread, error) {
	query := `INSERT INTO thread(
    slug,
    author,
    created,
    message,
    title,
	forum)
	VALUES (NULLIF($1, ''), $2, $3, $4, $5, $6) RETURNING *`

	forumObj, err := p.GetBySlug(thread.Forum)
	if err != nil {
		return models.Thread{}, err
	}

	var threadObj models.Thread
	if thread.Created != "" {
		err = p.conn.Get(&threadObj, query, thread.Slug, thread.Author,
			thread.Created, thread.Message, thread.Title, forumObj.Slug)
	} else {
		err = p.conn.Get(&threadObj, query, thread.Slug, thread.Author,
			time.Time{}, thread.Message, thread.Title, forumObj.Slug)
	}
	return threadObj, err
}

func (p *postgresForumRepository) GetThreads(slug string, limit int, since string, desc bool) ([]models.Thread, error) {
	var whereExpression string
	var orderExpression string

	if since != "" && desc {
		whereExpression = fmt.Sprintf(`LOWER(forum)=LOWER('%s') AND created <= '%s'`, slug, since)
	} else if since != "" && !desc {
		whereExpression = fmt.Sprintf(`LOWER(forum)=LOWER('%s') AND created >= '%s'`, slug, since)
	} else {
		whereExpression = fmt.Sprintf(`LOWER(forum)=LOWER('%s')`, slug)
	}
	if desc {
		orderExpression = `DESC`
	} else {
		orderExpression = `ASC`
	}

	query := fmt.Sprintf("SELECT * FROM thread WHERE %s ORDER BY created %s LIMIT NULLIF(%d, 0)",
		whereExpression, orderExpression, limit)

	var data []models.Thread
	err := p.conn.Select(&data, query)
	return data, err
}

func (p *postgresForumRepository) CheckThreadExists(slug string) (bool, error) {
	query := `select exists(select 1 from thread where forum=$1)`

	var exists bool

	err := p.conn.Get(&exists, query, slug)
	return exists, err
}

func (p *postgresForumRepository) GetThreadBySlug(slug string) (models.Thread, error) {
	query := `SELECT * FROM thread WHERE LOWER(slug)=LOWER($1)`

	var thread models.Thread

	err := p.conn.Get(&thread, query, slug)
	return thread, err
}

func (p *postgresForumRepository) GetThreadByID(id int) (models.Thread, error) {
	query := `SELECT * FROM thread WHERE id=$1`

	var thread models.Thread

	err := p.conn.Get(&thread, query, id)
	return thread, err
}

func (p *postgresForumRepository) GetThreadIDBySlug(slug string) (int, error) {
	query := `SELECT id FROM thread WHERE LOWER(slug)=LOWER($1)`

	var id int

	err := p.conn.Get(&id, query, slug)
	return id, err
}

func (p *postgresForumRepository) GetThreadSlugByID(id int) (string, error) {
	query := `SELECT slug FROM thread WHERE id=$1`
	//TODO INDEX
	var slug string
	err := p.conn.Get(&slug, query, id)
	return slug, err
}

func (p *postgresForumRepository) getForumSlug(threadID int) (string, error) {
	query := `SELECT forum FROM thread WHERE id=$1`

	var slug string
	err := p.conn.Get(&slug, query, threadID)
	return slug, err
}

func (p *postgresForumRepository) AddPosts(posts []models.Post, threadID int) ([]models.Post, error) {
	query := `INSERT INTO post(
                 author,
                 created,
                 message,
                 parent,
				 thread,
				 forum) VALUES `
	data := make([]models.Post, 0, 0)
	if len(posts) == 0 {
		return data, nil
	}

	slug, err := p.getForumSlug(threadID)
	if err != nil {
		return data, err
	}

	timeCreated := time.Now()
	var valuesNames []string
	var values []interface{}
	i := 1
	for _, element := range posts {
		valuesNames = append(valuesNames, fmt.Sprintf(
			"($%d, $%d, $%d, nullif($%d, 0), $%d, $%d)",
			i, i+1, i+2, i+3, i+4, i+5))
		i += 6
		values = append(values, element.Author, timeCreated, element.Message, element.Parent, threadID, slug)
	}

	query += strings.Join(valuesNames[:], ",")
	query += " RETURNING *"
	err = p.conn.Select(&data, query, values...)
	return data, err
}

func (p *postgresForumRepository) AddVote(vote models.Vote) error {
	query := `INSERT INTO vote(
				nickname,  
				voice,     
				idThread)
				VALUES ($1, $2, NULLIF($3, 0)) RETURNING *`

	_, err := p.conn.Exec(query, vote.Nickname, vote.Voice, vote.IdThread)
	return err
}

func (p *postgresForumRepository) UpdateVote(vote models.Vote) error {
	query := `UPDATE vote SET voice=$1 WHERE LOWER(nickname) = LOWER($2) AND idThread = $3 AND voice<>$1`
	_, err := p.conn.Exec(query, vote.Voice, vote.Nickname, vote.IdThread)
	return err
}

func (p *postgresForumRepository) getPostsFlat(threadID, limit, since int,
	desc bool) ([]models.Post, error) {

	query := `SELECT * FROM post WHERE thread=$1 `

	if desc {
		if since > 0 {
			query += fmt.Sprintf("AND id < %d ", since)
		}
		query += `ORDER BY id DESC `
	} else {
		if since > 0 {
			query += fmt.Sprintf("AND id > %d ", since)
		}
		query += `ORDER BY id `
	}
	query += `LIMIT NULLIF($2, 0)`
	var posts []models.Post
	err := p.conn.Select(&posts, query, threadID, limit)
	return posts, err
}

func (p *postgresForumRepository) getPostsTree(threadID, limit, since int,
	desc bool) ([]models.Post, error) {
	var query string
	sinceQuery := ""
	if since != 0 {
		if desc {
			sinceQuery = `AND PATH < `
		} else {
			sinceQuery = `AND PATH > `
		}
		sinceQuery += fmt.Sprintf(`(SELECT path FROM post WHERE id = %d)`, since)
	}
	if desc {
		query = fmt.Sprintf(
			`SELECT * FROM post WHERE thread=$1 %s ORDER BY path DESC, id DESC LIMIT NULLIF($2, 0);`, sinceQuery)
	} else {
		query = fmt.Sprintf(
			`SELECT * FROM post WHERE thread=$1 %s ORDER BY path, id LIMIT NULLIF($2, 0);`, sinceQuery)
	}
	var posts []models.Post
	err := p.conn.Select(&posts, query, threadID, limit)
	return posts, err
}

func (p *postgresForumRepository) getPostsParentTree(threadID, limit, since int,
	desc bool) ([]models.Post, error) {
	var query string
	sinceQuery := ""
	if since != 0 {
		if desc {
			sinceQuery = `AND PATH[1] < `
		} else {
			sinceQuery = `AND PATH[1] > `
		}
		sinceQuery += fmt.Sprintf(`(SELECT path[1] FROM post WHERE id = %d)`, since)
	}

	parentsQuery := fmt.Sprintf(
		`SELECT id FROM post WHERE thread = $1 AND parent IS NULL %s`, sinceQuery)

	if desc {
		parentsQuery += `ORDER BY id DESC`
		if limit > 0 {
			parentsQuery += fmt.Sprintf(` LIMIT %d`, limit)
		}
		query = fmt.Sprintf(
			`SELECT * FROM post WHERE path[1] IN (%s) ORDER BY path[1] DESC, path, id;`, parentsQuery)
	} else {
		parentsQuery += `ORDER BY id`
		if limit > 0 {
			parentsQuery += fmt.Sprintf(` LIMIT %d`, limit)
		}
		query = fmt.Sprintf(
			`SELECT * FROM post WHERE path[1] IN (%s) ORDER BY path,id;`, parentsQuery)
	}
	var posts []models.Post
	err := p.conn.Select(&posts, query, threadID)
	return posts, err
}

func (p *postgresForumRepository) GetPosts(postSlugOrId models.Thread, limit, since int,
	sort string, desc bool) ([]models.Post, error) {
	var err error
	threadId := 0
	if postSlugOrId.Id <= 0 {
		threadId, err = p.GetThreadIDBySlug(postSlugOrId.Slug.String)
		if err != nil {
			return nil, err
		}
	} else {
		threadId = int(postSlugOrId.Id)
	}

	switch sort {
	case "flat":
		return p.getPostsFlat(threadId, limit, since, desc)
	case "tree":
		return p.getPostsTree(threadId, limit, since, desc)
	case "parent_tree":
		return p.getPostsParentTree(threadId, limit, since, desc)
	default:
		return nil, errors.New("THERE IS NO SORT WITH THIS NAME")
	}
}

func (p *postgresForumRepository) GetPost(id int, related []string) (map[string]interface{}, error) {
	query := `SELECT * FROM post WHERE id = $1;`
	var post models.Post
	err := p.conn.Get(&post, query, id)

	returnMap := map[string]interface{}{
		"post": post,
	}

	for _, relatedObj := range related {
		switch relatedObj {
		case "user":
			author, err := p.userRepo.GetByNick(post.Author)
			if err != nil {
				return returnMap, err
			}
			returnMap["author"] = author
		case "thread":
			thread, err := p.GetThreadByID(int(post.Thread))
			if err != nil {
				return returnMap, err
			}
			returnMap["thread"] = thread
		case "forum":
			forumObj, err := p.GetBySlug(post.Forum)
			if err != nil {
				return returnMap, err
			}
			returnMap["forum"] = forumObj
		}
	}

	return returnMap, err
}

func (p *postgresForumRepository) UpdatePost(newPost models.Post) (models.Post, error) {
	query := `UPDATE post SET message = $1, isEdited = true WHERE id = $2 RETURNING *;`

	oldPost, err := p.GetPost(int(newPost.Id), []string{})
	if err != nil {
		return models.Post{}, err
	}
	if oldPost["post"].(models.Post).Message == newPost.Message {
		return oldPost["post"].(models.Post), nil
	}

	if newPost.Message == "" {
		query := `SELECT * FROM post WHERE id = $1`
		var post models.Post
		err := p.conn.Get(&post, query, newPost.Id)
		return post, err
	}
	var post models.Post
	err = p.conn.Get(&post, query, newPost.Message, newPost.Id)
	return post, err
}

func (p *postgresForumRepository) UpdateThread(newThread models.Thread) (models.Thread, error) {
	query := `UPDATE thread SET message=COALESCE(NULLIF($1, ''), message), title=COALESCE(NULLIF($2, ''), title) WHERE `

	if newThread.Id > 0 {
		query += `id = $3 RETURNING *`
		var thread models.Thread
		err := p.conn.Get(&thread, query, newThread.Message, newThread.Title, newThread.Id)
		return thread, err
	} else {
		query += `slug = $3 RETURNING *`
		var thread models.Thread
		err := p.conn.Get(&thread, query, newThread.Message, newThread.Title, newThread.Slug)
		return thread, err
	}
}

func (p *postgresForumRepository) GetServiceStatus() (map[string]int, error) {
	query := `SELECT * FROM (SELECT COUNT(*) FROM forum) as fC, (SELECT COUNT(*) FROM post) as pC,
              (SELECT COUNT(*) FROM thread) as tC, (SELECT COUNT(*) FROM users) as uC;`

	a, err := p.conn.Query(query)
	if err != nil {
		return nil, err
	}

	if a.Next() {
		forumCount, postCount, threadCount, usersCount := 0, 0, 0, 0
		err := a.Scan(&forumCount, &postCount, &threadCount, &usersCount)
		if err != nil {
			return nil, err
		}
		return map[string]int{
			"forum":  forumCount,
			"post":   postCount,
			"thread": threadCount,
			"user":   usersCount,
		}, nil
	}
	return nil, errors.New("no info available")
}

func (p *postgresForumRepository) ClearDatabase() error {
	query := `TRUNCATE users, forum, thread, post, vote, users_forum;`

	_, err := p.conn.Exec(query)
	return err
}
