package models

import (
	"database/sql"
	"encoding/json"
	"github.com/jackc/pgtype"
)

type Forum struct {
	Posts   int64  `json:"posts"`
	Slug    string `json:"slug"`
	Threads int32  `json:"threads"`
	Title   string `json:"title"`
	User    string `json:"user"`
}

type Thread struct {
	Author  string         `json:"author"`
	Created string         `json:"created"`
	Forum   string         `json:"forum"`
	Id      int32          `json:"id"`
	Message string         `json:"message"`
	Slug    JsonNullString `json:"slug"`
	Title   string         `json:"title"`
	Votes   int32          `json:"votes"`
}

type JsonNullInt64 struct {
	sql.NullInt64
}

func (v JsonNullInt64) MarshalJSON() ([]byte, error) {
	if v.Valid {
		return json.Marshal(v.Int64)
	} else {
		return json.Marshal(nil)
	}
}

func (v *JsonNullInt64) UnmarshalJSON(data []byte) error {
	var x *int64
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil {
		v.Valid = true
		v.Int64 = *x
	} else {
		v.Valid = false
	}
	return nil
}

type JsonNullString struct {
	sql.NullString
}

func (v JsonNullString) MarshalJSON() ([]byte, error) {
	if v.Valid {
		return json.Marshal(v.String)
	} else {
		return json.Marshal(nil)
	}
}

func (v *JsonNullString) UnmarshalJSON(data []byte) error {
	var x *string
	if err := json.Unmarshal(data, &x); err != nil {
		return err
	}
	if x != nil {
		v.Valid = true
		v.String = *x
	} else {
		v.Valid = false
	}
	return nil
}

type Post struct {
	Author   string           `json:"author"`
	Created  string           `json:"created"`
	Forum    string           `json:"forum"`
	Id       int64            `json:"id"`
	IsEdited bool             `json:"isEdited"`
	Message  string           `json:"message"`
	Parent   JsonNullInt64    `json:"parent"`
	Thread   int32            `json:"thread"`
	Path     pgtype.Int8Array `json:"-"`
}

type Vote struct {
	Nickname string `json:"nickname"`
	Voice    int32  `json:"voice"`
	IdThread int64  `json:"-"`
}
