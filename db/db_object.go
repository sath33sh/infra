// This package provides the database abstraction.
package db

import (
	"github.com/couchbaselabs/gocb"
	"github.com/sath33sh/infra/log"
	"github.com/sath33sh/infra/util"
)

// Object type.
type ObjType string

// Object metadata.
type ObjMeta struct {
	Bucket BucketIndex
	Type   ObjType
	Id     string
}

func (meta ObjMeta) Key() string {
	return string(meta.Type) + ":" + meta.Id
}

// Object interface.
// Structs that can be written to database should implement this interface.
type Object interface {
	GetMeta() ObjMeta // get object meta data.
	SetType()         // Set object type.
}

// Validate object metadata.
func getValidMeta(obj Object) (meta ObjMeta, err error) {
	// Get metadata.
	meta = obj.GetMeta()

	// Validate.
	if len(meta.Type) == 0 ||
		len(meta.Id) == 0 ||
		int(meta.Bucket) >= len(Buckets) {
		log.Errorf("Invalid metadata: type %s, id %s, bucket %d", meta.Type, meta.Id, meta.Bucket)
		return ObjMeta{}, util.ErrInvalidObject
	}

	return meta, nil
}

// Get object from database.
func Get(obj Object) error {
	// Validate metadata.
	meta, err := getValidMeta(obj)
	if err != nil {
		return err
	}

	// Get document from couchbase.
	_, err = Buckets[meta.Bucket].couch.Get(meta.Key(), obj)
	if err != nil {
		return util.ErrNotFound
	}

	return err
}

// Upsert object in to database.
func Upsert(obj Object, expiry uint32) error {
	// Set object type.
	obj.SetType()

	// Validate metadata.
	meta, err := getValidMeta(obj)
	if err != nil {
		return err
	}

	key := meta.Key()

	// Upsert document in couchbase.
	_, err = Buckets[meta.Bucket].couch.Upsert(key, obj, expiry)
	if err != nil {
		log.Errorf("%s Upsert() error: key %s: %v", Buckets[meta.Bucket].name, key, err)
		return util.ErrDbAccess
	}

	return err
}

// Remove object from database.
func Remove(obj Object) error {
	// Validate metadata.
	meta, err := getValidMeta(obj)
	if err != nil {
		return err
	}

	key := meta.Key()

	// Get and lock document before remove.
	var v interface{}
	cas, err := Buckets[meta.Bucket].couch.GetAndLock(key, LOCK_INTERVAL, &v)
	if err != nil {
		log.Errorf("%s GetAndLock() error: key %s: %v", Buckets[meta.Bucket].name, key, err)
		return util.ErrDbAccess
	}

	// Remove document from couchbase.
	_, err = Buckets[meta.Bucket].couch.Remove(key, cas)
	if err != nil {
		log.Errorf("%s Remove() error: key %s: %v", Buckets[meta.Bucket].name, key, err)
		return util.ErrDbAccess
	}

	return err
}

// Get and lock document.
func GetLock(obj Object) (Lock, error) {
	// Validate metadata.
	meta, err := getValidMeta(obj)
	if err != nil {
		return Lock(0), err
	}

	key := meta.Key()

	// Get and lock in couchbase.
	var cas gocb.Cas
	cas, err = Buckets[meta.Bucket].couch.GetAndLock(key, LOCK_INTERVAL, obj)
	if err != nil {
		log.Errorf("%s GraphGetLock() error: key %s: %v", Buckets[meta.Bucket].name, key, err)
		return Lock(cas), util.ErrNotFound
	}

	return Lock(cas), err
}

// Unlock.
func Unlock(obj Object, lock Lock) error {
	// Validate metadata.
	meta, err := getValidMeta(obj)
	if err != nil {
		return err
	}

	key := meta.Key()

	// Write and unlock in couchbase.
	_, err = Buckets[meta.Bucket].couch.Unlock(key, gocb.Cas(lock))
	if err != nil {
		log.Errorf("%s Unlock() error: key %s: %v", Buckets[meta.Bucket].name, key, err)
		return util.ErrDbAccess
	}

	return err
}

// Write and unlock.
func WriteUnlock(obj Object, lock Lock, expiry uint32) error {
	// Set object type just in case.
	obj.SetType()

	// Validate metadata.
	meta, err := getValidMeta(obj)
	if err != nil {
		return err
	}

	key := meta.Key()

	// Write and unlock in couchbase.
	_, err = Buckets[meta.Bucket].couch.Replace(key, obj, gocb.Cas(lock), expiry)
	if err != nil {
		log.Errorf("%s Replace() error: key %s: %v", Buckets[meta.Bucket].name, key, err)
		return util.ErrDbAccess
	}

	return err
}

// Perform multi-get from database. Returns number of successful gets.
func GetMulti(objs []Object) (nGets int, err error) {
	if len(objs) == 0 {
		// Nothing to do.
		return 0, nil
	}

	// Validate metadata.
	meta, err := getValidMeta(objs[0])
	if err != nil {
		return 0, err
	}

	/*
		// Setup couchbase bulk ops.
		ops := make([]gocb.GetOp, len(docs))
		opPtrs := make([]gocb.BulkOp, len(docs))
		for index, op := range ops {
			_, docType, key := docs[index].GetMetadata()
			op.Key = string(docType) + ":" + key
			op.Value = docs[index]
			opPtrs[index] = &ops[index]
		}

		// Perform bulk ops.
		err = Buckets[bIndex].couch.Do(opPtrs)
		if err != nil {
			return util.ERR_DB_ACCESS
		}

		for index, op := range ops {
			log.Errorf("Index %d, Error %v", index, op.Err)
		}
	*/

	// XXX - Couchbase native bulk operation is not working yet.
	// Perform individual gets for now.
	for _, obj := range objs {
		key := obj.GetMeta().Key()

		// Get document from couchbase.
		_, getErr := Buckets[meta.Bucket].couch.Get(key, obj)
		if getErr != nil {
			// log.Errorf("Failed to get %s, index %d: %v", key, index, getErr)
			err = getErr
		} else {
			nGets++
		}
	}

	return nGets, err
}
