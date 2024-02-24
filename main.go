package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"encoding/base64"
	"math/rand"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	sse "github.com/r3labs/sse/v2"
)

// generator a function type that returns string.
type generator func() string

var (
	random = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
)

type Account struct {
	Acct string `json:"acct"`
}

type Data struct {
	Content        string    `json:"content"`
	AccountDetails Account   `json:"account"`
	CreatedAt      time.Time `json:"created_at"`
}

type Comms struct {
	clientConnections map[string]chan Data
	postChannel       chan Data
	deleteChannel     chan string
}
type Streamer struct {
	postChannel   chan Data
	deleteChannel chan string
	connectionId  string
}

func WithCustomHeader(key, value string) func(c *sse.Client) {
	return func(c *sse.Client) {
		c.Headers[key] = value
	}
}

func genID(len int) string {
	bytes := make([]byte, len)
	random.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)[:len]
}

const (
	ConnectionId     = "connectionId"
	PostChannelKey   = "generalPost"
	DeleteChannelKey = "generalPost"
	StreamerKey      = "streamerKey"
)

func createClientConnections(comms *Comms) gin.HandlerFunc {
	log.Println("Comms initializing.")
	return func(ctx *gin.Context) {
		postChannel := make(chan Data)
		connectionId := genID(16)
		comms.clientConnections[connectionId] = postChannel
		streamerStruct := Streamer{postChannel: postChannel, deleteChannel: comms.deleteChannel, connectionId: connectionId}
		log.Printf("Connecting %s", connectionId)
		ctx.Set(StreamerKey, streamerStruct)
		ctx.Next()
	}
}

func main() {
	var address string
	flag.StringVar(&address, "address", "0.0.0.0:7878", "port to run")
	flag.Parse()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	mastodonServerDomain := os.Getenv("MASTODON_SERVER_DOMAIN")
	if len(mastodonServerDomain) == 0 {
		log.Fatal("domain not configured")
	}
	mastodonBearerToken := os.Getenv("MASTODON_BEARER_TOKEN")
	if len(mastodonBearerToken) == 0 {
		log.Fatal("Token not configured")
	}

	postChannel := make(chan Data)
	deleteChannel := make(chan string)
	clientConnections := make(map[string]chan Data)
	comms := Comms{postChannel: postChannel, clientConnections: clientConnections, deleteChannel: deleteChannel}
	go getstream(comms.postChannel, mastodonServerDomain, mastodonBearerToken)
	go distributor(&comms)
	go remover(&comms)

	r := gin.Default()

	r.Use(createClientConnections(&comms))
	r.GET("/ping/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/stream", streamer)
	r.Run(address) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func getstream(postChannel chan<- Data, serverDomain string, bearerToken string) {

	// add authorization header to the req
	client := sse.NewClient(serverDomain, WithCustomHeader("Authorization", bearerToken))
	fmt.Println("Connected to stream")
	client.Subscribe("updated", func(msg *sse.Event) {
		// Got some data!
		var data Data
		json.Unmarshal(msg.Data, &data)
		postChannel <- data
	})
}

func distributor(comms *Comms) {
	log.Println("Launching distributor")
	for {
		select {
		case data, ok := <-comms.postChannel:
			if !ok {
				// Close the response writer when the channel is closed
				return
			}
			log.Printf("Got message from:%s", data.AccountDetails.Acct)

			for connectionId, individualChannel := range comms.clientConnections {
				log.Printf("Sending message from:%s to :%s", data.AccountDetails.Acct, connectionId)
				individualChannel <- data
			}
		}
	}

}
func remover(comms *Comms) {
	log.Println("Launching remover")
	for {
		select {
		case connectionId, ok := <-comms.deleteChannel:
			if !ok {
				// Close the response writer when the channel is closed
				return
			}
			log.Printf("Removing :%s", connectionId)
			delete(comms.clientConnections, connectionId)
		}
	}

}

func streamer(c *gin.Context) {
	// Get the response writer from the context
	streamerStruct, ok := c.Value(StreamerKey).(Streamer)
	if !ok {
		log.Fatal("Streamer not available in context")
	}

	for {
		select {
		case <-c.Request.Context().Done():
			streamerStruct.deleteChannel <- streamerStruct.connectionId

			return
		case data, ok := <-streamerStruct.postChannel:
			if !ok {
				// Close the response writer when the channel is closed
				return
			}

			// Write the incoming data to the response writer
			c.SSEvent("message", map[string]interface{}{
				"type": "data",
				"data": data,
			})
			// Flush the response to ensure the data is sent immediately
			c.Writer.Flush()
		}
	}

}
