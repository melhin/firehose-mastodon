package main

import (
	"encoding/json"
	managers "firehoseMastodon/managers"
	models "firehoseMastodon/models"
	"log"

	sse "github.com/r3labs/sse/v2"
)

func postStream(data []byte) Data {
	var streamData Data
	json.Unmarshal(data, &streamData)
	if len(streamData.Content) > 0 {
		inputPost := models.PostContent{
			Content:       streamData.Content,
			PostId:        streamData.Id,
			User:          streamData.AccountDetails.Acct,
			PostCreatedAt: streamData.CreatedAt,
		}
		models.CreatePost(inputPost)
	}
	return streamData
}

func PullFromSource(postChannel chan<- managers.TransferData, serverDomain string, bearerToken string) {

	client := sse.NewClient(serverDomain, WithCustomHeader("Authorization", bearerToken))
	log.Println("Connected to stream")
	err := client.SubscribeRaw(func(msg *sse.Event) {
		streamData := postStream(msg.Data)
		if len(streamData.Content) > 0 {
			transferData := managers.TransferData{Id: streamData.Id, Data: msg.Data}
			postChannel <- transferData
		}
	})
	if err != nil {
		log.Fatalf("Cannot connect to given domain:%s . Check env", serverDomain)
	}
	log.Println("DisConnected to stream")
}
