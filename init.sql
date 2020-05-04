CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE "users"
(
    About           text,
	Email           citext UNIQUE,
	FullName        text NOT NULL,
	Nickname        citext PRIMARY KEY
);

CREATE TABLE forum
(
    "user"    citext,
	Posts     BIGINT DEFAULT 0,
	Slug      citext PRIMARY KEY,
	Threads   INT DEFAULT 0,
	title     text,
	FOREIGN KEY ("user") REFERENCES "users" (nickname)
);

CREATE TABLE thread
(
    author    citext,
	created   timestamp with time zone default now(),
	forum     citext,
	id        SERIAL PRIMARY KEY,
	message   text NOT NULL,
	slug      citext UNIQUE,
	title     text not null,
	votes     INT default 0,
	FOREIGN KEY (author) REFERENCES "users" (nickname),
    FOREIGN KEY (forum) REFERENCES "forum" (slug)
);
CREATE OR REPLACE FUNCTION update_user_forum() RETURNS TRIGGER AS $update_users_forum$
    BEGIN
           INSERT INTO users_forum (nickname, Slug) VALUES (NEW.author, NEW.forum) on conflict do nothing;
           return NEW;
    end
    $update_users_forum$ LANGUAGE plpgsql;


CREATE TABLE post
(
    author    citext NOT NULL,
	created   timestamp with time zone default now(),
	forum     citext,
	id        BIGSERIAL PRIMARY KEY,
	isEdited  BOOLEAN DEFAULT FALSE,
    message   text NOT NULL,
	parent    BIGINT DEFAULT 0,
	thread    INT,
	path      BIGINT[] default array[]::INTEGER[],
	FOREIGN KEY (author) REFERENCES "users" (nickname),
    FOREIGN KEY (forum) REFERENCES "forum" (slug),
    FOREIGN KEY (thread) REFERENCES "thread" (id),
    FOREIGN KEY (parent) REFERENCES "post" (id)
);

CREATE OR REPLACE FUNCTION update_path() RETURNS TRIGGER AS $update_path$
DECLARE
    parent_path  BIGINT[];
    first_parent post;
BEGIN
    IF (NEW.parent IS NULL) THEN
        NEW.path := array_append(new.path, new.id);
    ELSE
        SELECT path FROM post WHERE id = new.parent INTO parent_path;
        SELECT * FROM post WHERE id = parent_path[1] INTO first_parent;
        IF NOT FOUND OR first_parent.thread != NEW.thread THEN
            RAISE EXCEPTION 'parent is from different thread' USING ERRCODE = '00409';
        end if;

        NEW.path := NEW.path || parent_path || new.id;
    end if;
    RETURN new;
end
$update_path$ LANGUAGE plpgsql;

CREATE TABLE vote
(
    nickname   citext NOT NULL,
	voice      INT,
	idThread   INT,

	FOREIGN KEY (nickname) REFERENCES "users" (nickname),
    FOREIGN KEY (idThread) REFERENCES "thread" (id),
    UNIQUE(nickname, idThread)
);


CREATE TABLE users_forum
(
    nickname citext NOT NULL,
    Slug     citext NOT NULL,
    FOREIGN KEY (nickname) REFERENCES "users" (nickname),
    FOREIGN KEY (Slug) REFERENCES "forum" (Slug),
    UNIQUE (nickname, Slug)
);

CREATE TRIGGER thread_insert_user_forum AFTER INSERT ON thread FOR EACH ROW EXECUTE PROCEDURE update_user_forum();
CREATE TRIGGER post_insert_user_forum AFTER INSERT ON post FOR EACH ROW EXECUTE PROCEDURE update_user_forum();
CREATE TRIGGER path_update_trigger BEFORE INSERT ON post FOR EACH ROW EXECUTE PROCEDURE update_path();