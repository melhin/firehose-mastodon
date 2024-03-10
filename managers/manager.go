package managers

import (
	"log"
	"time"

	"math/rand"
	"sync"

	"github.com/google/uuid"
)

var (
	random = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
)

type TransferData struct {
	Id   string
	Data []byte
}
type Comms struct {
	mu                sync.Mutex
	clientConnections map[uuid.UUID]chan TransferData
	postChannel       chan TransferData
}

func (comms *Comms) SetConnection(deleteChannel chan uuid.UUID) StreamerData {
	individualPostChannel := make(chan TransferData)
	connectionId := uuid.New()
	comms.mu.Lock()
	defer comms.mu.Unlock()
	comms.clientConnections[connectionId] = individualPostChannel
	return StreamerData{IndividualPostChannel: individualPostChannel, DeleteChannel: deleteChannel, ConnectionId: connectionId}
}
func (comms *Comms) DeleteConnection(connectionId uuid.UUID) {
	comms.mu.Lock()
	delete(comms.clientConnections, connectionId)
	comms.mu.Unlock()
}

type StreamerData struct {
	IndividualPostChannel chan TransferData
	DeleteChannel         chan uuid.UUID
	ConnectionId          uuid.UUID
}

func NewComms(PostChannel chan TransferData, deleteChannel chan uuid.UUID) Comms {
	clientConnections := make(map[uuid.UUID]chan TransferData)
	comms := Comms{postChannel: PostChannel, clientConnections: clientConnections}
	go Distributor(&comms)
	go Remover(&comms, deleteChannel)
	return comms
}

func Distributor(comms *Comms) {
	log.Println("Launching distributor")
	for {
		select {
		case data, ok := <-comms.postChannel:
			if !ok {
				// Close the response writer when the channel is closed
				return
			}
			log.Printf("Got message from:%s", data.Id)
			comms.mu.Lock()

			for connectionId, individualChannel := range comms.clientConnections {
				log.Printf("Sending message from:%s to :%s", data.Id, connectionId)
				individualChannel <- data
			}
			comms.mu.Unlock()
		}
	}

}
func Remover(comms *Comms, deleteChannel chan uuid.UUID) {
	log.Println("Launching remover")
	for {
		select {
		case connectionId, ok := <-deleteChannel:
			if !ok {
				// Close the response writer when the channel is closed
				return
			}
			log.Printf("Removing :%s", connectionId)
			comms.DeleteConnection(connectionId)
		}
	}

}
