package push

import (
	"encoding/json"
	"fmt"
	"github.com/sath33sh/infra/config"
	"github.com/sath33sh/infra/log"
	"os"
	"sync"
	"testing"
	"time"
)

// Pushable test object.
type testObject struct {
	uri  string
	data string
}

func (obj *testObject) BuildPushPayload() (*Payload, error) {
	p := &Payload{
		Kind: "test",
		Op:   UPSERT,
		Uri:  obj.uri,
		Data: json.RawMessage(obj.data),
	}

	return p, nil
}

type clientSpawner struct {
	readyWg      sync.WaitGroup // Waitgroup for client readiness.
	doneWg       sync.WaitGroup // Waitgroup for done.
	numClients   int            // Number of clients to spawn.
	numMsgs      int            // Number of messages expected by clients.
	waitInterval time.Duration  // Time to wait for messages.
	userId       string         // User ID.
	topicUri     string         // Topic URI.
}

func NewUserClientSpawner(userId string, numClients, numMsgs int, waitInterval time.Duration) *clientSpawner {
	return &clientSpawner{
		numClients:   numClients,
		numMsgs:      numMsgs,
		waitInterval: waitInterval,
		userId:       userId,
	}
}

func NewTopicClientSpawner(topicUri string, numClients, numMsgs int, waitInterval time.Duration) *clientSpawner {
	return &clientSpawner{
		numClients:   numClients,
		numMsgs:      numMsgs,
		waitInterval: waitInterval,
		topicUri:     topicUri,
	}
}

func (cs *clientSpawner) mockClient(t *testing.T, inst int) {
	var userId, sessionId string
	var subscribe bool

	if len(cs.userId) == 0 {
		// Topic test.
		userId = fmt.Sprintf("%d", inst)
		sessionId = userId
		subscribe = true
	} else {
		// User test.
		userId = cs.userId
		sessionId = fmt.Sprintf("%d", inst)
	}

	//t.Logf("Start client: userId %s, sessionId %s\n", userId, sessionId)

	duct := OpenSession(userId, sessionId, true)
	timer := time.NewTimer(cs.waitInterval)

	// Subscribe to topic.
	if subscribe {
		Subscribe(cs.topicUri, userId, sessionId, true)
	}

	// Ready to receive.
	cs.readyWg.Done()

	for i := 0; i < cs.numMsgs; i++ {
		// t.Logf("Expect %d\n", i)
		select {
		case <-duct:
			// t.Logf("Receive %d\n", i)
			continue

		case <-timer.C:
			t.Fatalf("Client %s:%s timed out", userId, sessionId)
		}
	}

	//t.Logf("End client: userId %s, sessionId %s\n", userId, sessionId)

	// Cleanup
	if subscribe {
		Unsubscribe(cs.topicUri, userId, sessionId)
	}
	CloseSession(userId, sessionId, duct)
	cs.doneWg.Done()
}

func (cs *clientSpawner) Spawn(t *testing.T) {
	// Initialize waitgroups.
	cs.readyWg.Add(cs.numClients)
	cs.doneWg.Add(cs.numClients)

	for i := 1; i <= cs.numClients; i++ {
		go cs.mockClient(t, i)
	}

	// Wait for client readiness.
	cs.readyWg.Wait()
}

func (cs *clientSpawner) Wait(t *testing.T) {
	cs.doneWg.Wait()
}

func TestPushToUser(t *testing.T) {
	testUserId := "100"
	numMsgs := 1000

	// Spawn client.
	cs := NewUserClientSpawner(testUserId, 1, numMsgs, 5*time.Second)
	cs.Spawn(t)

	// Push messages.
	for i := 0; i < numMsgs; i++ {
		PushToUser(testUserId, &testObject{uri: "testuri", data: "This is a test"})
	}

	// Wait.
	cs.Wait(t)
}

func TestPublish(t *testing.T) {
	testUri := "test:uri"
	numMsgs := 1000

	// Spawn clients.
	cs := NewTopicClientSpawner(testUri, 10, numMsgs, 5*time.Second)
	cs.Spawn(t)

	// Push messages.
	for i := 0; i < numMsgs; i++ {
		Publish(&testObject{uri: testUri, data: "This is a test"})
	}

	// Wait.
	cs.Wait(t)
}

func TestMain(m *testing.M) {
	// Init.
	config.Init()
	log.Init("")
	Init()

	os.Exit(m.Run())
}
