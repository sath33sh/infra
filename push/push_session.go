package push

import (
	"github.com/sath33sh/infra/log"
	"sync"
)

// Session command.
type SessionCmdType int

const (
	ONLINE SessionCmdType = iota
	OFFLINE
)

// Constants.
const (
	CMD_DUCT_BUFFER_MAX  = 200
	DATA_DUCT_BUFFER_MAX = 200
)

// Session Key.
type SessionKey string

// Session.
type Session struct {
	payloadDuct chan *Payload // Channel for sending payload to client.
	msgsSent    uint          // Number of messages sent to this session.
}

// Session command.
type SessionCmd struct {
	cmd         SessionCmdType // Command type.
	userId      string         // User ID.
	sessionId   string         // Session ID.
	payloadDuct chan *Payload  // Client's payload duct.
	signalDone  bool           // Signal command completion.
	wg          sync.WaitGroup // Waitgroup for signaling completion.
}

// Online sessions.
var sessions struct {
	sync.RWMutex                                    // Mutex for accessing online users.
	users        map[string]map[SessionKey]*Session // Map of online user ID to set of session data.
	cmdDuct      chan *SessionCmd                   // Channel for sending commands to session manager.
}

func sessionMgrLoop() {
	for {
		select {
		case sc := <-sessions.cmdDuct:
			skey := SessionKey(sc.userId + ":" + sc.sessionId)

			log.Debugf(MODULE, "Command %d: session %s", sc.cmd, skey)

			switch sc.cmd {
			case ONLINE:
				// Lock sessions.
				sessions.Lock()

				// Create user entry if it does not exist.
				if _, ok := sessions.users[sc.userId]; !ok {
					// User entry does not exist. Create.
					sessions.users[sc.userId] = make(map[SessionKey]*Session)
				}

				// Add or update session.
				sessions.users[sc.userId][skey] = &Session{
					payloadDuct: sc.payloadDuct,
				}

				// Unlock sessions.
				sessions.Unlock()

				// Signal done.
				if sc.signalDone {
					sc.wg.Done()
				}

			case OFFLINE:
				// Lock sessions.
				sessions.Lock()

				if _, ok := sessions.users[sc.userId]; ok {
					// Delete session.
					if es, ok := sessions.users[sc.userId][skey]; ok {
						// Delete session only if duct pointer matches.
						// Otherwise we are deleting the wrong session.
						if sc.payloadDuct == es.payloadDuct {
							delete(sessions.users[sc.userId], skey)
						}
					}

					if len(sessions.users[sc.userId]) == 0 {
						// Delete user entry.
						delete(sessions.users, sc.userId)
					}
				}

				// Unlock sessions.
				sessions.Unlock()

			default:
				log.Errorf("Invalid command %d", sc.cmd)
			}
		}
	}
}

// Start session manager.
func startSessionMgr() {
	// Initialize sessions.
	sessions.users = make(map[string]map[SessionKey]*Session)
	sessions.cmdDuct = make(chan *SessionCmd, CMD_DUCT_BUFFER_MAX)

	// Start session manager loop.
	go sessionMgrLoop()
}

func OpenSession(userId string, sessionId string, wait bool) chan *Payload {
	// Make data duct for the client.
	duct := make(chan *Payload, DATA_DUCT_BUFFER_MAX)

	cmd := &SessionCmd{
		cmd:         ONLINE,
		userId:      userId,
		sessionId:   sessionId,
		payloadDuct: duct,
	}

	if wait {
		cmd.signalDone = true
		cmd.wg.Add(1)
	}

	// Send online command to session manager.
	sessions.cmdDuct <- cmd

	if wait {
		// Wait for command completion.
		cmd.wg.Wait()
	}

	return duct
}

func lookupSession(userId string, sessionId string) (s *Session) {
	skey := SessionKey(userId + ":" + sessionId)

	// Lock sessions.
	sessions.RLock()

	if _, ok := sessions.users[userId]; ok {
		if _, ok = sessions.users[userId][skey]; ok {
			s = sessions.users[userId][skey]
		}
	}

	// Unlock sessions.
	sessions.RUnlock()

	return s
}

func CloseSession(userId string, sessionId string, duct chan *Payload) {
	// Unscribe session from all topics.
	unsubscribeAll(userId, sessionId)

	// Send offline command to session manager.
	sessions.cmdDuct <- &SessionCmd{
		cmd:         OFFLINE,
		userId:      userId,
		sessionId:   sessionId,
		payloadDuct: duct,
	}

	// Close payload duct.
	close(duct)

	return
}

func PushToUser(userId string, obj Pushable) (err error) {
	// Acquire read lock.
	sessions.RLock()

	if len(sessions.users[userId]) > 0 {
		// Build payload and push it to user sessions.
		var p *Payload
		if p, err = obj.BuildPushPayload(); err == nil {
			for _, s := range sessions.users[userId] {
				s.payloadDuct <- p
			}
		}
	}

	// Release read lock.
	sessions.RUnlock()

	return err
}
