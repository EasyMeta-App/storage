package mongo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Collection mongo-driver collection
type Collection struct {
	collection *mongo.Collection
}

// Index creates an index with the given keys and options.
func (c *Collection) Index(keys ...bson.M) ([]string, error) {
	ctx := context.Background()

	// 检查是否有唯一索引
	curIndex, err := c.collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}

	// 已经存在的索引
	indexes := map[string]bool{}
	defer curIndex.Close(ctx)
	for curIndex.Next(ctx) {
		var index bson.M
		curIndex.Decode(&index)

		keys := []string{}
		for k, v := range index["key"].(bson.M) {
			keys = append(keys, fmt.Sprintf("%v:%v", k, v))
		}

		key := strings.Join(keys, "_")
		if key == "" {
			key = index["name"].(string)
		}
		if _, ok := indexes[key]; !ok {
			indexes[key] = true
		}
	}

	var newIndexes []mongo.IndexModel
	for _, val := range keys {
		// get keys
		keys := []string{}
		if vs, ok := val["keys"].(bson.D); ok {
			for _, v := range vs {
				keys = append(keys, fmt.Sprintf("%v:%v", v.Key, v.Value))
			}
		}
		if vs, ok := val["keys"].(bson.M); ok {
			for k, v := range vs {
				keys = append(keys, fmt.Sprintf("%v:%v", k, v))
			}
		}
		key := strings.Join(keys, "_")
		if _, ok := indexes[key]; ok {
			continue
		}
		opts := options.Index().SetName(key)
		if v, ok := val["unique"]; ok && v.(bool) {
			opts.SetUnique(true)
		}
		newIndexes = append(newIndexes, mongo.IndexModel{
			Keys:    val["keys"],
			Options: opts,
		})
	}

	if len(newIndexes) == 0 {
		return nil, nil
	}
	return c.collection.Indexes().CreateMany(ctx, newIndexes)
}

// Where finds docs by given filter
func (c *Collection) Where(filter bson.D) *Session {
	return &Session{filter: filter, collection: c.collection, table: c}
}

// FindByID finds a single document by id.
func (c *Collection) FindByID(id primitive.ObjectID, result interface{}) error {
	return c.Where(bson.D{{"_id", id}}).Find(result)
}

// InsertWithResult inserts a single document into the collection and returns insert one result.
func (c *Collection) Insert(doc interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	return c.collection.InsertOne(context.Background(), doc, opts...)
}

// InsertAllWithResult inserts the provided documents and returns insert many result.
func (c *Collection) InsertAll(docs []interface{}, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	return c.collection.InsertMany(context.Background(), docs, opts...)
}

// Aggregate performs an aggregation pipeline.
func (c *Collection) Aggregate(pipeline interface{}, results interface{}, opts ...*options.AggregateOptions) error {
	// 设置超时时间
	ao := options.MergeAggregateOptions(opts...)
	maxTime := 10 * time.Second
	if ao.MaxTime != nil {
		maxTime = *ao.MaxTime
	}
	ctx, cancel := context.WithTimeout(context.Background(), maxTime)
	defer cancel()
	cur, err := c.collection.Aggregate(ctx, pipeline, ao)
	if err != nil {
		return err
	}
	return decode(ctx, cur, results)
}
