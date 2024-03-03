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
type PostContent struct {
	Content       string
	User          string
	PostId        string
	PostCreatedAt time.Time
}

type Account struct {
	Acct string `json:"acct"`
}
type OutputPosts struct {
	Content        string    `json:"content"`
	AccountDetails Account   `json:"account"`
	CreatedAt      time.Time `json:"created_at"`
	Id             string    `json:"id"`
}

type OutputData struct {
	OutputPosts   []OutputPosts
	LastTimeStamp time.Time
}

func CreatePost(inputPost PostContent) {
	post := Post{Content: inputPost.Content, User: inputPost.User, PostId: &inputPost.PostId, PostCreatedAt: inputPost.PostCreatedAt}
	DB.Create(&post)
	log.Printf("Post stored id:%d, created_at:%s", post.ID, post.CreatedAt)
}

func GetLatestMessages(lastTimeStamp time.Time) (OutputData, error) {
	var posts []Post
	var currentLastTimeStamp time.Time
	currentLastTimeStamp = lastTimeStamp
	result := DB.Where("created_at > ?", currentLastTimeStamp).Order("created_At").Find(&posts)
	if result.Error != nil {
		return OutputData{}, result.Error
	}
	var targetData []OutputPosts
	if len(posts) == 0 {
		return OutputData{}, nil
	}
	for _, post := range posts {
		target := OutputPosts{AccountDetails: Account{Acct: post.User}, Content: post.Content, CreatedAt: post.PostCreatedAt, Id: *post.PostId}
		targetData = append(targetData, target)
		currentLastTimeStamp = post.CreatedAt
	}

	return OutputData{OutputPosts: targetData, LastTimeStamp: currentLastTimeStamp}, nil
}
