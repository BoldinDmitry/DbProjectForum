ALTER SYSTEM SET
    max_connections = '300';
ALTER SYSTEM SET
    shared_buffers = '225MB';
ALTER SYSTEM SET
    effective_cache_size = '675MB';
ALTER SYSTEM SET
    maintenance_work_mem = '57600kB';
ALTER SYSTEM SET
    checkpoint_completion_target = '0.9';
ALTER SYSTEM SET
    wal_buffers = '6912kB';
ALTER SYSTEM SET
    default_statistics_target = '100';
ALTER SYSTEM SET
    random_page_cost = '1.1';
ALTER SYSTEM SET
    effective_io_concurrency = '200';
ALTER SYSTEM SET
    work_mem = '768kB';
ALTER SYSTEM SET
    min_wal_size = '2GB';
ALTER SYSTEM SET
    max_wal_size = '8GB';
ALTER SYSTEM SET
    max_worker_processes = '2';
ALTER SYSTEM SET
    max_parallel_workers_per_gather = '1';
ALTER SYSTEM SET
    max_parallel_workers = '2';

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE "users"
(
    About    text,
    Email    citext UNIQUE,
    FullName text NOT NULL,
    Nickname citext PRIMARY KEY
);

CREATE TABLE forum
(
    "user"  citext,
    Posts   BIGINT DEFAULT 0,
    Slug    citext PRIMARY KEY,
    Threads INT    DEFAULT 0,
    title   text,
    FOREIGN KEY ("user") REFERENCES "users" (nickname)
);

CREATE TABLE thread
(
    author  citext,
    created timestamp with time zone default now(),
    forum   citext,
    id      SERIAL PRIMARY KEY,
    message text NOT NULL,
    slug    citext UNIQUE,
    title   text not null,
    votes   INT                      default 0,
    FOREIGN KEY (author) REFERENCES "users" (nickname),
    FOREIGN KEY (forum) REFERENCES "forum" (slug)
);
CREATE OR REPLACE FUNCTION update_user_forum() RETURNS TRIGGER AS
$update_users_forum$
BEGIN
    INSERT INTO users_forum (nickname, Slug) VALUES (NEW.author, NEW.forum) on conflict do nothing;
    return NEW;
end
$update_users_forum$ LANGUAGE plpgsql;


CREATE TABLE post
(
    author   citext NOT NULL,
    created  timestamp with time zone default now(),
    forum    citext,
    id       BIGSERIAL PRIMARY KEY,
    isEdited BOOLEAN                  DEFAULT FALSE,
    message  text   NOT NULL,
    parent   BIGINT                   DEFAULT 0,
    thread   INT,
    path     BIGINT[]                 default array []::INTEGER[],
    FOREIGN KEY (author) REFERENCES "users" (nickname),
    FOREIGN KEY (forum) REFERENCES "forum" (slug),
    FOREIGN KEY (thread) REFERENCES "thread" (id),
    FOREIGN KEY (parent) REFERENCES "post" (id)
);

CREATE OR REPLACE FUNCTION update_path() RETURNS TRIGGER AS
$update_path$
DECLARE
    parent_path         BIGINT[];
    first_parent_thread INT;
BEGIN
    IF (NEW.parent IS NULL) THEN
        NEW.path := array_append(new.path, new.id);
    ELSE
        SELECT path FROM post WHERE id = new.parent INTO parent_path;
        SELECT thread FROM post WHERE id = parent_path[1] INTO first_parent_thread;
        IF NOT FOUND OR first_parent_thread != NEW.thread THEN
            RAISE EXCEPTION 'parent is from different thread' USING ERRCODE = '00409';
        end if;

        NEW.path := NEW.path || parent_path || new.id;
    end if;
    UPDATE forum SET Posts=Posts + 1 WHERE lower(forum.slug) = lower(new.forum);
    RETURN new;
end
$update_path$ LANGUAGE plpgsql;

CREATE TABLE vote
(
    nickname citext NOT NULL,
    voice    INT,
    idThread INT,

    FOREIGN KEY (nickname) REFERENCES "users" (nickname),
    FOREIGN KEY (idThread) REFERENCES "thread" (id),
    UNIQUE (nickname, idThread)
);


CREATE TABLE users_forum
(
    nickname citext NOT NULL,
    Slug     citext NOT NULL,
    FOREIGN KEY (nickname) REFERENCES "users" (nickname),
    FOREIGN KEY (Slug) REFERENCES "forum" (Slug),
    UNIQUE (nickname, Slug)
);

CREATE OR REPLACE FUNCTION insert_votes() RETURNS TRIGGER AS
$update_users_forum$
BEGIN
    UPDATE thread SET votes=(votes + NEW.voice) WHERE id = NEW.idThread;
    return NEW;
end
$update_users_forum$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_votes() RETURNS TRIGGER AS
$update_users_forum$
BEGIN
    UPDATE thread SET votes=(votes + NEW.voice * 2) WHERE id = NEW.idThread;
    return NEW;
end
$update_users_forum$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_threads_count() RETURNS TRIGGER AS
$update_users_forum$
BEGIN
    UPDATE forum SET Threads=(Threads + 1) WHERE LOWER(slug) = LOWER(NEW.forum);
    return NEW;
end
$update_users_forum$ LANGUAGE plpgsql;

CREATE TRIGGER thread_insert_user_forum
    AFTER INSERT
    ON thread
    FOR EACH ROW
EXECUTE PROCEDURE update_user_forum();

CREATE TRIGGER post_insert_user_forum
    AFTER INSERT
    ON post
    FOR EACH ROW
EXECUTE PROCEDURE update_user_forum();

CREATE TRIGGER path_update_trigger
    BEFORE INSERT
    ON post
    FOR EACH ROW
EXECUTE PROCEDURE update_path();

CREATE TRIGGER add_vote
    BEFORE INSERT
    ON vote
    FOR EACH ROW
EXECUTE PROCEDURE insert_votes();

CREATE TRIGGER add_thread_to_forum
    BEFORE INSERT
    ON thread
    FOR EACH ROW
EXECUTE PROCEDURE update_threads_count();

CREATE TRIGGER edit_vote
    BEFORE UPDATE
    ON vote
    FOR EACH ROW
EXECUTE PROCEDURE update_votes();

CREATE INDEX post_first_parent_thread_index ON post ((post.path[1]), thread);
CREATE INDEX post_first_parent_id_index ON post ((post.path[1]), id);
CREATE INDEX post_first_parent_index ON post ((post.path[1]));
CREATE INDEX post_path_index ON post ((post.path));
CREATE INDEX post_thread_index ON post (thread); -- -
CREATE INDEX post_thread_id_index ON post (thread, id); -- +

CREATE INDEX forum_slug_lower_index ON forum (lower(forum.Slug)); -- +

CREATE INDEX users_nickname_lower_index ON users (lower(users.Nickname));
CREATE INDEX users_nickname_index ON users ((users.Nickname));
CREATE INDEX users_email_index ON users (lower(Email));

CREATE INDEX users_forum_forum_user_index ON users_forum (lower(users_forum.Slug), nickname);
CREATE INDEX users_forum_user_index ON users_forum (nickname);

CREATE INDEX thread_slug_lower_index ON thread (lower(slug));
CREATE INDEX thread_slug_index ON thread (slug);
CREATE INDEX thread_slug_id_index ON thread (lower(slug), id);
CREATE INDEX thread_forum_lower_index ON thread (lower(forum)); -- +
CREATE INDEX thread_id_forum_index ON thread (id, forum);
CREATE INDEX thread_created_index ON thread (created);

CREATE INDEX vote_nickname ON vote (lower(nickname), idThread, voice);
-- +

-- NEW INDEXES
CREATE INDEX post_path_id_index ON post (id, (post.path));
CREATE INDEX post_thread_path_id_index ON post (thread, (post.parent), id);

CREATE INDEX users_forum_forum_index ON users_forum ((users_forum.Slug)); -- +
