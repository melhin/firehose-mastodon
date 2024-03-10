package main

import (
	"encoding/json"
	managers "firehoseMastodon/managers"
	models "firehoseMastodon/models"
	"log"

	sse "github.com/r3labs/sse/v2"
)

func postStream(msg *sse.Event) (Data, bool) {

	var streamData Data
	var contentPresent bool
	json.Unmarshal(msg.Data, &streamData)
	if len(streamData.Content) > 0 {
		inputPost := models.PostContent{
			Content:       streamData.Content,
			PostId:        streamData.Id,
			User:          streamData.AccountDetails.Acct,
			PostCreatedAt: streamData.CreatedAt,
		}
		models.CreatePost(inputPost)
		contentPresent = true
	}
	return streamData, contentPresent
}

func PullFromSource(postChannel chan<- managers.TransferData, serverDomain string, bearerToken string) {

	// add authorization header to the req
	client := sse.NewClient(serverDomain, WithCustomHeader("Authorization", bearerToken))
	log.Println("Connected to stream")
	err := client.SubscribeRaw(func(msg *sse.Event) {
		streamData, contentPresent := postStream((msg))
		if contentPresent {

			transferData := managers.TransferData{Id: streamData.Id, Data: msg.Data}
			postChannel <- transferData
		}
	})
	if err != nil {
		log.Fatalf("Cannot connect to given domain:%s . Check env", serverDomain)
	}
	log.Println("DisConnected to stream")
}
