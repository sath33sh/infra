package db

import (
	"fmt"
	"github.com/couchbaselabs/gocb"
	"github.com/sath33sh/infra/log"
	"github.com/sath33sh/infra/util"
	"strconv"
)

// Query result interface.
type QueryResult interface {
	GetRowPtr(int) interface{}
}

// Query limits.
const (
	QUERY_LIMIT_DEFAULT = 20
	QUERY_LIMIT_MAX     = 200
)

// Parse query page arguments limit and offset.
func ParsePageArgs(limitStr, offsetStr string) (limit, offset int, err error) {
	// Parse limit.
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			log.Errorf("Invalid limit %s", limitStr)
			return 0, 0, util.ErrInvalidInput
		}
	}

	if limit == 0 {
		limit = QUERY_LIMIT_DEFAULT
	} else if limit > QUERY_LIMIT_MAX {
		limit = QUERY_LIMIT_MAX
	}

	// Parse offset.
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			log.Errorf("Invalid offset %s", offsetStr)
			return 0, 0, util.ErrInvalidInput
		}
	}

	return limit, offset, nil
}

// Execute N1QL query.
func ExecQuery(bIndex BucketIndex, qr QueryResult, queryStmt string) (size int, err error) {
	log.Debugf(MODULE, "Bucket %d, Query {%s}", bIndex, queryStmt)

	// Execute query.
	q := gocb.NewN1qlQuery(queryStmt)
	r, err := Buckets[bIndex].couch.ExecuteN1qlQuery(q, nil)
	if err != nil {
		log.Errorf("N1QL query error: stmt %s: %v", queryStmt, err)
		return size, util.ErrDbAccess
	}

	// Save results.
	for r.Next(qr.GetRowPtr(size)) {
		size++
	}

	err = r.Close()
	if err != nil {
		log.Errorf("N1QL query close error: stmt %s: %v", queryStmt, err)
		return size, util.ErrDbAccess
	}

	return size, nil
}

// Execute N1QL query with pagination.
func ExecPagedQuery(bIndex BucketIndex, qr QueryResult, queryStmt string, limit, offset int) (size int, err error) {

	log.Debugf(MODULE, "Bucket %d, Query {%s}, limit %d, offset %d", bIndex, queryStmt, limit, offset)

	// Add limit and offset to query statement.
	queryStmt += fmt.Sprintf(" limit %d", limit)
	if offset > 0 {
		queryStmt += fmt.Sprintf(" offset %d", offset)
	}

	// Execute query.
	q := gocb.NewN1qlQuery(queryStmt)
	r, err := Buckets[bIndex].couch.ExecuteN1qlQuery(q, nil)
	if err != nil {
		log.Errorf("N1QL query error: stmt %s: %v", queryStmt, err)
		return size, util.ErrDbAccess
	}

	// Save results.
	for r.Next(qr.GetRowPtr(size)) {
		size++
	}

	err = r.Close()
	if err != nil {
		log.Errorf("N1QL query close error: stmt %s: %v", queryStmt, err)
		return size, util.ErrDbAccess
	}

	return size, nil
}

// Count query result.
type CountResult struct {
	Count int `json:"$1"`
}

// Execute count N1QL query.
func ExecCount(bIndex BucketIndex, queryStmt string) (int, error) {
	log.Debugf(MODULE, "Bucket %d, Query {%s}", bIndex, queryStmt)

	// Execute query.
	q := gocb.NewN1qlQuery(queryStmt)
	r, err := Buckets[bIndex].couch.ExecuteN1qlQuery(q, nil)
	if err != nil {
		log.Errorf("N1QL query error: stmt %s: %v", queryStmt, err)
		return 0, util.ErrDbAccess
	}

	// Get result count.
	var cr CountResult
	r.One(&cr)

	err = r.Close()
	if err != nil {
		log.Errorf("N1QL query close error: stmt %s: %v", queryStmt, err)
		return 0, util.ErrDbAccess
	}

	return cr.Count, nil
}

// View query result.
const (
	VIEW_QUERY_LIMIT_DEFAULT = 20
	VIEW_QUERY_LIMIT_MAX     = 200
)

type ViewResult struct {
	Id string `json:"id"` // Document ID.
}

type ViewQueryResult struct {
	Results    []ViewResult `json:"results,omitempty"` // Results.
	NextOffset string       `json:"nextOffset"`        // Next offset.
	PrevOffset string       `json:"prevOffset"`        // Previous offset.
}

func (qr *ViewQueryResult) MakeRows(size int) int {
	if size == 0 {
		size = VIEW_QUERY_LIMIT_DEFAULT
	} else if size > VIEW_QUERY_LIMIT_MAX {
		size = VIEW_QUERY_LIMIT_MAX
	}

	if len(qr.Results) < size {
		qr.Results = make([]ViewResult, size)
	}

	return size
}

func (qr *ViewQueryResult) GetRowPtr(index int) interface{} {
	if index < len(qr.Results) {
		return &qr.Results[index]
	} else {
		return nil
	}
}

// Execute view query.
func ExecPagedViewQuery(
	bIndex BucketIndex,
	qr QueryResult,
	designDoc, viewName,
	key,
	limitStr, offsetStr string) (size, offset int, err error) {
	var limit int

	log.Debugf(MODULE, "Bucket %d, view %s:%s, key %s, limit %s, offset %s",
		bIndex, designDoc, viewName, key, limitStr, offsetStr)

	// Validate limit.
	if len(limitStr) > 0 {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			log.Errorf("Invalid limit %s", limitStr)
			return 0, 0, util.ErrInvalidInput
		}
	}

	// Validate offset.
	if len(offsetStr) > 0 {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			log.Errorf("Invalid offset %s", offsetStr)
			return 0, 0, util.ErrInvalidInput
		}
	}

	// Execute query.
	q := gocb.NewViewQuery(designDoc, viewName).Skip(uint(offset)).Limit(uint(limit)).Order(gocb.Descending)
	if key != "" {
		q = q.Key(key)
	}
	r, err := Buckets[bIndex].couch.ExecuteViewQuery(q)
	if err != nil {
		log.Errorf("View query error: %s:%s: %v", designDoc, viewName, err)
		return size, offset, util.ErrDbAccess
	}

	// Save results.
	for r.Next(qr.GetRowPtr(size)) {
		size++
	}

	err = r.Close()
	if err != nil {
		log.Errorf("View query close error: %s:%s: %v", designDoc, viewName, err)
		return size, offset, util.ErrDbAccess
	}

	return size, offset, nil
}

// Execute view query with start and end keys.
func ExecPagedViewQueryInRange(
	bIndex BucketIndex,
	qr QueryResult,
	designDoc, viewName string,
	startKey, endKey interface{},
	limitStr, offsetStr string) (size, offset int, err error) {
	var limit int

	log.Debugf(MODULE, "Bucket %d, view %s:%s, limit %s, offset %s",
		bIndex, designDoc, viewName, limitStr, offsetStr)

	// Validate limit.
	if len(limitStr) > 0 {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			log.Errorf("Invalid limit %s", limitStr)
			return 0, 0, util.ErrInvalidInput
		}
	}

	// Validate offset.
	if len(offsetStr) > 0 {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			log.Errorf("Invalid offset %s", offsetStr)
			return 0, 0, util.ErrInvalidInput
		}
	}

	// Execute query.
	q := gocb.NewViewQuery(designDoc, viewName).Skip(uint(offset)).
		Range(startKey, endKey, true).
		Limit(uint(limit)).Order(gocb.Descending)
	r, err := Buckets[bIndex].couch.ExecuteViewQuery(q)
	if err != nil {
		log.Errorf("View query error: %s:%s: %v", designDoc, viewName, err)
		return size, offset, util.ErrDbAccess
	}

	// Save results.
	for r.Next(qr.GetRowPtr(size)) {
		size++
	}

	err = r.Close()
	if err != nil {
		log.Errorf("View query close error: %s:%s: %v", designDoc, viewName, err)
		return size, offset, util.ErrDbAccess
	}

	return size, offset, nil
}
