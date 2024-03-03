package main

import (
	"flag"
	"log"
	"net/http"

	"encoding/json"
	"os"
	"time"

	managers "firehoseMastodon/managers"
	models "firehoseMastodon/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sse "github.com/r3labs/sse/v2"
)

// generator a function type that returns string.
type generator func() string

const (
	StreamerKey = "streamerKey"
)

type Account struct {
	Acct string `json:"acct"`
}
type Data struct {
	Content        string    `json:"content"`
	AccountDetails Account   `json:"account"`
	CreatedAt      time.Time `json:"created_at"`
	Id             string    `json:"id"`
}

func WithCustomHeader(key, value string) func(c *sse.Client) {
	return func(c *sse.Client) {
		c.Headers[key] = value
	}
}

func createClientConnections(comms *managers.Comms, deleteChannel chan uuid.UUID) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		streamerData := comms.SetConnection(deleteChannel)
		log.Printf("Connecting %s", streamerData.ConnectionId)
		ctx.Set(StreamerKey, streamerData)
		ctx.Next()
	}
}

func main() {
	var address string
	flag.StringVar(&address, "address", "0.0.0.0:8000", "port to run")
	flag.Parse()
	mastodonServerDomain := os.Getenv("MASTODON_SERVER_DOMAIN")
	if len(mastodonServerDomain) == 0 {
		log.Fatal("domain not configured")
	}
	mastodonBearerToken := os.Getenv("MASTODON_BEARER_TOKEN")
	if len(mastodonBearerToken) == 0 {
		log.Fatal("Token not configured")
	}
	dbName := os.Getenv("DB_NAME")
	if len(mastodonBearerToken) == 0 {
		log.Fatal("DB name not provided")
	}

	models.ConnectDatabase(dbName)
	models.Migrate()

	postChannel := make(chan managers.TransferData)
	deleteChannel := make(chan uuid.UUID)
	storageChannel := make(chan managers.TransferData)

	go getstream(postChannel, storageChannel, mastodonServerDomain, mastodonBearerToken)
	go storePost(storageChannel)
	comms := managers.NewComms(postChannel, deleteChannel)

	r := gin.Default()

	r.Use(createClientConnections(&comms, deleteChannel))
	r.GET("/ping/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/stream/", Streamer)
	r.Run(address) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func getstream(postChannel chan<- managers.TransferData, storageChannel chan<- managers.TransferData, serverDomain string, bearerToken string) {

	// add authorization header to the req
	client := sse.NewClient(serverDomain, WithCustomHeader("Authorization", bearerToken))
	err := client.SubscribeRaw(func(msg *sse.Event) {
		log.Println("Connected to stream")
		var streamData Data
		json.Unmarshal(msg.Data, &streamData)
		transferData := managers.TransferData{Id: streamData.Id, Data: msg.Data}
		postChannel <- transferData
		storageChannel <- transferData
	})
	if err != nil {
		log.Fatalf("Cannot connect to given domain:%s . Check env", serverDomain)
	}
	log.Println("DisConnected to stream")
}

func storePost(storageChannel chan managers.TransferData) {
	for {
		select {
		case data, ok := <-storageChannel:
			if !ok {
				// Close the response writer when the channel is closed
				return
			}

			var streamData Data
			json.Unmarshal(data.Data, &streamData)
			inputPost := models.InputPost{Content: streamData.Content, PostId: streamData.Id, User: streamData.AccountDetails.Acct, PostCreatedAt: streamData.CreatedAt}
			models.CreatePost(inputPost)
		}
	}

}

func Streamer(c *gin.Context) {
	// Get the response writer from the context
	streamerStruct, ok := c.Value(StreamerKey).(managers.StreamerData)
	if !ok {
		log.Fatal("Streamer not available in context")
	}

	for {
		select {
		case <-c.Request.Context().Done():
			streamerStruct.DeleteChannel <- streamerStruct.ConnectionId

			return
		case data, ok := <-streamerStruct.IndividualPostChannel:
			if !ok {
				// Close the response writer when the channel is closed
				return
			}
			// Got some data!
			var streamData Data
			json.Unmarshal(data.Data, &streamData)

			// Write the incoming data to the response writer
			c.SSEvent("message", map[string]interface{}{
				"type": "data",
				"data": streamData,
			})
			// Flush the response to ensure the data is sent immediately
			c.Writer.Flush()
		}
	}

}
