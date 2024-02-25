package managers

import (
	"log"
)

type TransferData struct {
	Id   string
	Data []byte
}
type Comms struct {
	clientConnections map[string]chan TransferData
	postChannel       chan TransferData
}

func (comms *Comms) SetConnection(connectionId string, postChannel chan TransferData) bool {
	comms.clientConnections[connectionId] = postChannel
	return true
}
func (comms *Comms) DeleteConnection(connectionId string) bool {
	delete(comms.clientConnections, connectionId)
	return true
}

type StreamerData struct {
	IndividualPostChannel chan TransferData
	DeleteChannel         chan string
	ConnectionId          string
}

func NewComms(PostChannel chan TransferData, deleteChannel chan string) Comms {
	clientConnections := make(map[string]chan TransferData)
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

			for connectionId, individualChannel := range comms.clientConnections {
				log.Printf("Sending message from:%s to :%s", data.Id, connectionId)
				individualChannel <- data
			}
		}
	}

}
func Remover(comms *Comms, deleteChannel chan string) {
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
