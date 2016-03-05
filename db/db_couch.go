package db

import (
	"github.com/couchbaselabs/gocb"
	"github.com/sath33sh/infra/config"
	"github.com/sath33sh/infra/log"
	"github.com/sath33sh/infra/util"
	"time"
	//gocb "gopkg.in/couchbaselabs/gocb.v0"
)

// Module name.
const MODULE = "db"

// Integer constants.
const (
	LOCK_INTERVAL = 30 // Seconds.
)

// Lock.
type Lock gocb.Cas

// Bucket index.
type BucketIndex int

const (
	DEFAULT_BUCKET BucketIndex = iota // Default bucket.
)

// Bucket.
type bucket struct {
	index BucketIndex  // Bucket index.
	name  string       // Bucket name.
	couch *gocb.Bucket // Couchbase bucket.
}

// Array of buckets.
var Buckets = [...]bucket{
	bucket{index: DEFAULT_BUCKET},
}

// Local variables.
var (
	spec    string        // Connection spec.
	cluster *gocb.Cluster // Couchbase cluster.
)

func Init() {
	// Debug enable.
	log.EnableDebug(MODULE)

	// Get connection spec from config file.
	spec = config.Base.GetString("db-couch", "spec", "")
	if spec != "" {
		log.Infoln("Couchbase connection spec:", spec)
	} else {
		log.Fatalf("Couchbase connection spec not found")
	}

	var err error
	cluster, err = gocb.Connect(spec)
	if err != nil {
		log.Fatalf("Couchbase Connect() error: host %s: %v", spec, err)
	}

	// Open buckets.
	Buckets[DEFAULT_BUCKET].open("default")
}

// Open bucket.
func (b *bucket) open(name string) (err error) {
	b.name = name
	b.couch, err = cluster.OpenBucket(b.name, "")
	if err != nil {
		log.Fatalf("%s OpenBucket() error: host %s: %v", b.name, spec, err)
	}

	return err
}

// Get bucket name given the bucket index.
func BucketName(index BucketIndex) string {
	return Buckets[index].name
}

// Counter.
func (b *bucket) Counter(key string, delta, initial int64, expiry uint32) (uint64, error) {
	newval, _, err := b.couch.Counter(key, delta, initial, expiry)
	if err != nil {
		log.Errorf("%s Counter() error: key %s: %v", b.name, key, err)
		return 0, util.ErrDbAccess
	}

	return newval, err
}

// Calculate document expiry from number of days.
func CalcExpiry(days int) uint32 {
	return uint32(time.Now().Unix() + int64(days*24*int(time.Hour/time.Second)))
}
