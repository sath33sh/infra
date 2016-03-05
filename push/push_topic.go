package push

import (
	"github.com/sath33sh/infra/log"
	"sync"
	"time"
)

// Command types.
type TopicCmdType int

const (
	SUBSCRIBE   TopicCmdType = iota // Subscribe to a topic.
	UNSUBSCRIBE                     // Unsubscribe from a topic.
	CLEAR                           // Unsubscribe from all subscribed topics.
	STOP                            // Stop topic loop.
)

// Topic command.
type TopicCmd struct {
	cmd        TopicCmdType   // Command.
	uri        string         // Topic URI.
	userId     string         // User ID.
	sessionId  string         // Session ID.
	signalDone bool           // Signal command completion.
	wg         sync.WaitGroup // Waitgroup for signaling completion.
}

// Topic.
type Topic struct {
	sync.RWMutex                         // Mutex for accessing topic structure.
	subscribers  map[SessionKey]*Session // Set of subscribers.
	payloadDuct  chan *Payload           // Channel for sending payload to topic.
	cmdDuct      chan *TopicCmd          // Channel for sending topic commands.
}

// Online topics.
var topics struct {
	sync.RWMutex                                 // Mutex for accessing online topics.
	topics        map[string]*Topic              // Set of topic pointers.
	subscriptions map[SessionKey]map[string]bool // Topic subscriptions.
	cmdDuct       chan *TopicCmd                 // Channel for sending commands to topic manager.
}

func (t *Topic) Loop(uri string) {
	log.Debugf(MODULE, "Enter topic loop %s", uri)

	for {
		select {
		case tc := <-t.cmdDuct:
			// Process command.
			skey := SessionKey(tc.userId + ":" + tc.sessionId)

			log.Debugf(MODULE, "Topic %s: command %d: session %s", uri, tc.cmd, skey)

			switch tc.cmd {
			case SUBSCRIBE:
				// Lock topic.
				t.Lock()

				// Add subscriber.
				if s := lookupSession(tc.userId, tc.sessionId); s != nil {
					t.subscribers[skey] = s
				} else {
					log.Errorf("Session %s not found", skey)
				}

				// Unlock topic.
				t.Unlock()

				// Signal done.
				if tc.signalDone {
					tc.wg.Done()
				}

			case CLEAR:
				fallthrough
			case UNSUBSCRIBE:
				// Lock topic.
				t.Lock()

				// Remove subscriber.
				delete(t.subscribers, skey)

				// Unlock topic.
				t.Unlock()

			case STOP:
				log.Debugf(MODULE, "Stop topic loop %s", uri)

				// Close channels and return.
				close(t.payloadDuct)
				close(t.cmdDuct)

				return

			default:
				log.Errorf("Invalid command %d", tc.cmd)
			}

		case payload := <-t.payloadDuct:
			// Process data.
			//log.Debugf(MODULE, "Topic %s, data %s", payload.Uri, payload.Data)

			// Acquire read lock.
			t.RLock()

			for _, s := range t.subscribers {
				s.payloadDuct <- payload
			}

			// Release read lock.
			t.RUnlock()
		}
	}
}

func startTopic(uri string) *Topic {
	t := &Topic{
		subscribers: make(map[SessionKey]*Session),
		payloadDuct: make(chan *Payload, DATA_DUCT_BUFFER_MAX),
		cmdDuct:     make(chan *TopicCmd, CMD_DUCT_BUFFER_MAX),
	}

	go t.Loop(uri)

	return t
}

func topicMgrLoop() {
	const CleanupTime = 24 * time.Hour

	cleanupTicker := time.NewTicker(CleanupTime)

	for {
		select {
		case tc := <-topics.cmdDuct:
			skey := SessionKey(tc.userId + ":" + tc.sessionId)

			log.Debugf(MODULE, "Command %d: uri %s, session %s", tc.cmd, tc.uri, skey)

			switch tc.cmd {
			case SUBSCRIBE:
				// Lock topics.
				topics.Lock()

				// Start topic worker if it doesn't exist.
				topic, exists := topics.topics[tc.uri]
				if !exists {
					topic = startTopic(tc.uri)

					topics.topics[tc.uri] = topic
				}

				// Forward subscribe command to topic.
				topic.cmdDuct <- tc

				// Update subscriptions.
				if _, ok := topics.subscriptions[skey]; !ok {
					topics.subscriptions[skey] = make(map[string]bool)
				}
				topics.subscriptions[skey][tc.uri] = true

				// Unlock topics.
				topics.Unlock()

			case UNSUBSCRIBE:
				// Lock topics.
				topics.Lock()

				// Forward unsubscribe command to topic, if it exists.
				if topic, exists := topics.topics[tc.uri]; exists {
					// Forward unsubscribe command to topic.
					topic.cmdDuct <- tc
				}

				// Update subscriptions.
				if _, ok := topics.subscriptions[skey]; ok {
					delete(topics.subscriptions[skey], tc.uri)

					if len(topics.subscriptions[skey]) == 0 {
						delete(topics.subscriptions, skey)
					}
				}

				// Unlock topics.
				topics.Unlock()

			case CLEAR:
				// Lock topics.
				topics.Lock()

				for uri, _ := range topics.subscriptions[skey] {
					if topic, exists := topics.topics[uri]; exists {
						// Send unsubscribe command to topic.
						topic.cmdDuct <- tc
					}

					delete(topics.subscriptions[skey], uri)
				}

				// Clear session.
				delete(topics.subscriptions, skey)

				// Unlock topics.
				topics.Unlock()

			default:
				log.Errorf("Invalid command %d", tc.cmd)
			}

		case <-cleanupTicker.C:
			// Lock topics.
			topics.Lock()

			for uri, topic := range topics.topics {
				topic.RLock()
				if len(topic.subscribers) == 0 {
					// No more subscribers. Stop the topic.
					topic.cmdDuct <- &TopicCmd{
						cmd: STOP,
					}

					// Delete topic.
					delete(topics.topics, uri)
				}
				topic.RUnlock()
			}

			log.Debugf(MODULE, "Cleanup: %d active topics", len(topics.topics))

			// Unlock topics.
			topics.Unlock()
		}
	}
}

// Start topic manager.
func startTopicMgr() {
	// Initialize sessions.
	topics.topics = make(map[string]*Topic)
	topics.subscriptions = make(map[SessionKey]map[string]bool)
	topics.cmdDuct = make(chan *TopicCmd, CMD_DUCT_BUFFER_MAX)

	// Start topic manager loop.
	go topicMgrLoop()
}

func Subscribe(uri string, userId string, sessionId string, wait bool) {
	cmd := &TopicCmd{
		cmd:       SUBSCRIBE,
		uri:       uri,
		userId:    userId,
		sessionId: sessionId,
	}

	if wait {
		cmd.signalDone = true
		cmd.wg.Add(1)
	}

	// Send subscribe command to topic manager.
	topics.cmdDuct <- cmd

	if wait {
		// Wait for command completion.
		cmd.wg.Wait()
	}
}

func Unsubscribe(uri string, userId string, sessionId string) {
	// Send unsubscribe command to topic manager.
	topics.cmdDuct <- &TopicCmd{
		cmd:       UNSUBSCRIBE,
		uri:       uri,
		userId:    userId,
		sessionId: sessionId,
	}
}

func unsubscribeAll(userId string, sessionId string) {
	// Send clear command to topic manager.
	topics.cmdDuct <- &TopicCmd{
		cmd:       CLEAR,
		userId:    userId,
		sessionId: sessionId,
	}
}

func processEgress(p *Payload) error {
	if !CasMode {
		return nil
	}

	// Get topic.
	topics.RLock()
	topic, ok := topics.topics[p.Uri]
	topics.RUnlock()

	if ok {
		// Topic exists. Send to topic worker.
		topic.payloadDuct <- p
	}

	return nil
}

func Publish(obj Pushable) error {
	// Build payload.
	p, err := obj.BuildPushPayload()
	if err != nil {
		return err
	}

	if DisableBroker {
		return processEgress(p)
	} else {
		return doPublishToBroker(p)
	}
}
