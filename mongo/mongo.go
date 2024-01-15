package mongo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/EasyMeta-App/helper"
	"go.mongodb.org/mongo-driver/mongo"
)

// ErrMongoNoDoc is mongo: no document results
var ErrMongoNoDoc = errors.New("mongo: no document results")

type (
	BulkWriteResult         = mongo.BulkWriteResult
	InsertOneResult         = mongo.InsertOneResult
	InsertManyResult        = mongo.InsertManyResult
	DeleteResult            = mongo.DeleteResult
	RewrapManyDataKeyResult = mongo.RewrapManyDataKeyResult
	ListDatabasesResult     = mongo.ListDatabasesResult
	DatabaseSpecification   = mongo.DatabaseSpecification
	UpdateResult            = mongo.UpdateResult
	IndexSpecification      = mongo.IndexSpecification
	CollectionSpecification = mongo.CollectionSpecification
	Pipeline                = mongo.Pipeline
)

// New session
//
// Relevant documentation:
//
//	https://docs.mongodb.com/manual/reference/connection-string/
func New(uri ...string) *Session {
	URI := helper.Config("MONGO_URI")
	if len(uri) > 0 {
		URI = uri[0]
	}
	session := &Session{
		uri: URI,
	}
	if err := session.Connect(); err != nil {
		log.Panic(err)
	}
	// 检查，如果失败，则重试
	go func(session *Session) {
		ticker := time.NewTicker(10 * time.Second)
		for ; true; <-ticker.C {
			if err := session.Ping(); err != nil {
				// 失败，重试
				if err := session.Connect(); err != nil {
					log.Println(err)
				}
			}
		}
	}(session)

	return session
}

var instance *Session = nil

// single mode for session
func Get(uri ...string) *Session {
	if instance != nil {
		return instance
	}
	instance = New(uri...)
	return instance
}

// C Collection alias
func C(collection string) *Collection {
	return Get().Collection(collection)
}

// decode
func decode(ctx context.Context, cur *mongo.Cursor, results interface{}) error {
	resultsVal := reflect.ValueOf(results)
	if resultsVal.Kind() != reflect.Ptr {
		return fmt.Errorf("results argument must be a pointer to a slice, but was a %s", resultsVal.Kind())
	}

	sliceVal := resultsVal.Elem()
	if sliceVal.Kind() == reflect.Interface {
		sliceVal = sliceVal.Elem()
	}

	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("results argument must be a pointer to a slice, but was a pointer to %s", sliceVal.Kind())
	}

	elementType := sliceVal.Type().Elem()

	defer cur.Close(ctx)
	var index int
	for cur.Next(ctx) {
		data := reflect.New(elementType)
		if err := cur.Decode(data.Interface()); err != nil {
			return err
		}
		sliceVal.Set(reflect.Append(sliceVal, data.Elem()))
		index++
	}

	if err := cur.Err(); err != nil {
		return err
	}

	resultsVal.Elem().Set(sliceVal.Slice(0, index))
	return nil
}
