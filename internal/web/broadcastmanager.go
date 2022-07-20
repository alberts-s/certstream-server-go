package web

import (
	"go-certstream-server/internal/certstream"
	"log"
	"sync"
)

type BroadcastManager struct {
	Broadcast  chan certstream.Entry
	clients    []*client
	clientLock sync.RWMutex
}

// registerClient adds a client to the list of clients of the BroadcastManager.
// The client will receive certificate broadcasts right after registration.
func (bm *BroadcastManager) registerClient(c *client) {
	bm.clientLock.Lock()
	bm.clients = append(bm.clients, c)
	log.Printf("Clients: %d, Capacity: %d\n", len(bm.clients), cap(bm.clients))
	bm.clientLock.Unlock()
}

// unregisterClient removes a client from the list of clients of the BroadcastManager.
// The client will no longer receive certificate broadcasts right after unregistering.
func (bm *BroadcastManager) unregisterClient(c *client) {
	bm.clientLock.Lock()
	for i, client := range bm.clients {
		if c == client {
			// Copy the last element of the slice to the position of the removed element
			// Then remove the last element by reslicing
			bm.clients[i] = bm.clients[len(bm.clients)-1]
			bm.clients[len(bm.clients)-1] = nil
			bm.clients = bm.clients[:len(bm.clients)-1]
			break
		}
	}
	log.Printf("Clients: %d, Capacity: %d\n", len(bm.clients), cap(bm.clients))
	bm.clientLock.Unlock()
}

// broadcaster is run in a goroutine and handles the dispatching of entries to clients.
func (bm *BroadcastManager) broadcaster() {
	for {
		entry := <-bm.Broadcast
		dataLite := entry.JSONLite()
		dataFull := entry.JSON()

		bm.clientLock.RLock()
		for _, client := range bm.clients {
			if client.fullStream {
				select {
				case client.broadcastChan <- dataFull:
				default:
					log.Printf("Skipping client '%s' because it's full\n", client.name)
				}
				continue
			}
			select {
			case client.broadcastChan <- dataLite:
			default:
				log.Printf("Skipping client '%s' because it's full\n", client.name)
			}
		}
		bm.clientLock.RUnlock()
	}
}
