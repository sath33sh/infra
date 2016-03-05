package push

import (
	"github.com/nats-io/nats"
	"github.com/sath33sh/infra/config"
	"github.com/sath33sh/infra/log"
	"github.com/sath33sh/infra/util"
)

// Nats client.
type NatsClient struct {
	opts  nats.Options
	conn  *nats.Conn
	econn *nats.EncodedConn
}

// Global variables.
var (
	natsClient = NatsClient{opts: nats.DefaultOptions}
)

func initNats() error {
	// Check whether broker is disabled.
	DisableBroker = config.Base.GetBool("push-nats", "disable", false)
	if DisableBroker {
		log.Infoln("Push broker disabled")
		return nil
	}

	// Read server URLs from config.
	natsClient.opts.Servers = config.Base.GetStringSlice("push-nats", "servers", []string{"nats://localhost:4222"})

	// Connect to broker.
	var err error
	natsClient.conn, err = natsClient.opts.Connect()
	if err != nil {
		log.Errorf("Failed to connect to push broker: %v", err)
		return util.ErrNetAccess
	}

	natsClient.econn, err = nats.NewEncodedConn(natsClient.conn, nats.JSON_ENCODER)
	if err != nil {
		log.Errorf("Failed to attach JSON encoder: %v", err)
		return util.ErrNetAccess
	}

	// Disconnect callback.
	natsClient.conn.Opts.DisconnectedCB = func(_ *nats.Conn) {
		log.Errorf("Disconnected from push broker")
	}

	// Reconnect callback.
	natsClient.conn.Opts.ReconnectedCB = func(nc *nats.Conn) {
		log.Errorf("Reconnected to push broker")
	}

	return nil
}

func processPayloadFromBroker(p *Payload) {
	// log.Debugf(MODULE, "Rx from broker: Kind %s, Uri %s, Op %s", p.Kind, p.Uri, p.Op)

	// Process.
	processEgress(p)
}

func SubscribeFromBroker(kinds []string) {
	if DisableBroker {
		return
	}

	for _, kind := range kinds {
		natsClient.econn.Subscribe(kind, processPayloadFromBroker)
	}
}

func doPublishToBroker(p *Payload) error {
	// Publish.
	natsClient.econn.Publish(p.Kind, p)

	return nil
}

func PublishToBroker(p *Payload) error {
	if DisableBroker {
		// Broker is disabled. Mock success.
		return nil
	}

	return doPublishToBroker(p)
}
