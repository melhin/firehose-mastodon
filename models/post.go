package models

import (
	"log"
	"time"
)

type Post struct {
	Content       string
	User          string
	PostId        *string
	PostCreatedAt time.Time
	ID            uint `json:"id" gorm:"primary_key"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
type InputPost struct {
	Content       string
	User          string
	PostId        string
	PostCreatedAt time.Time
}

func CreatePost(inputPost InputPost) {
	post := Post{Content: inputPost.Content, User: inputPost.User, PostId: &inputPost.PostId, PostCreatedAt: inputPost.PostCreatedAt}
	DB.Create(&post)
	log.Printf("Post stored id:%d, created_at:%s", post.ID, post.CreatedAt)
}
