package mongo

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

// Session mongo session
type Session struct {
	client     *mongo.Client
	collection *mongo.Collection
	table      *Collection
	db         string
	uri        string
	m          sync.RWMutex
	filter     bson.D
	findOpts   []*options.FindOptions
}

// C Collection alias
func (s *Session) C(collection string) *Collection {
	return s.Collection(collection)
}

// Collection returns collection
func (s *Session) Collection(collection string) *Collection {
	s.m.Lock()
	defer s.m.Unlock()
	if len(s.db) == 0 {
		s.db = "test"
	}
	d := &Database{database: s.client.Database(s.db)}
	return &Collection{collection: d.database.Collection(collection)}
}

// Connect mongo client
func (s *Session) Connect() error {
	cs, err := connstring.Parse(s.uri)
	if err != nil {
		return err
	}

	timeout := cs.ConnectTimeout
	if timeout == 0 {
		// 连接超时
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(s.uri))
	if err != nil {
		return err
	}

	s.client = client
	s.db = cs.Database
	return nil
}

// Ping verifies that the client can connect to the topology.
// If readPreference is nil then will use the client's default read
// preference.
func (s *Session) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	return s.client.Ping(ctx, readpref.Primary())
}

// Client return mongo Client
func (s *Session) Client() *mongo.Client {
	return s.client
}

// DB returns a value representing the named database.
func (s *Session) DB(db string) *Database {
	return &Database{database: s.client.Database(db)}
}

// SetOpts set find options
func (s *Session) SetOpts(opts ...*options.FindOptions) *Session {
	s.findOpts = opts
	return s
}

// Find returns up to one document that matches the model.
func (s *Session) Find(result interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	data, err := s.collection.FindOne(ctx, s.filter).DecodeBytes()
	if err != nil {
		return err
	}
	return bson.Unmarshal(data, result)
}

// FetchAll find all
func (s *Session) FetchAll(results interface{}) error {
	// 设置超时时间
	ctx := context.Background()
	fo := options.MergeFindOptions(s.findOpts...)
	if fo.NoCursorTimeout == nil || !*fo.NoCursorTimeout {
		maxTime := 10 * time.Second
		if fo.MaxTime != nil {
			maxTime = *fo.MaxTime
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, maxTime)
		defer cancel()
	}

	cur, err := s.collection.Find(ctx, s.filter, s.findOpts...)
	if err != nil {
		return err
	}

	return decode(ctx, cur, results)
}

// Update by id
func (s *Session) UpdateID(id primitive.ObjectID, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return s.collection.UpdateOne(context.Background(), bson.D{{"_id", id}}, update, opts...)
}

// Update one
func (s *Session) UpdateOne(update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	if s.filter == nil {
		s.filter = bson.D{}
	}
	return s.collection.UpdateOne(context.Background(), s.filter, update, opts...)
}

// Update all
func (s *Session) Update(update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	if s.filter == nil {
		s.filter = bson.D{}
	}
	return s.collection.UpdateMany(context.Background(), s.filter, update, opts...)
}

// Remove by ID
func (s *Session) RemoveID(id primitive.ObjectID, opts ...*options.DeleteOptions) error {
	s.filter = bson.D{{"_id", id}}
	return s.Remove(opts...)
}

// Remove
func (s *Session) Remove(opts ...*options.DeleteOptions) error {
	if s.filter == nil {
		return errors.New("filter is nil")
	}
	if _, err := s.collection.DeleteMany(context.Background(), s.filter, opts...); err != nil {
		return err
	}
	return nil
}

// Remove one
func (s *Session) RemoveOne(opts ...*options.DeleteOptions) error {
	if s.filter == nil {
		return errors.New("filter is nil")
	}
	if _, err := s.collection.DeleteOne(context.Background(), s.filter, opts...); err != nil {
		return err
	}
	return nil
}

// Count gets the number of documents matching the filter.
func (s *Session) Count(opts ...*options.CountOptions) int64 {
	if s.filter == nil {
		s.filter = bson.D{}
	}
	if v, err := s.collection.CountDocuments(context.Background(), s.filter, opts...); err == nil {
		return v
	}
	return 0
}

// Pagination pagination
func (s *Session) Pagination(page, limit int, results interface{}) (int64, error) {
	fo := options.MergeFindOptions(s.findOpts...)
	if limit > 0 {
		fo.SetLimit(int64(limit))
		offset := (page - 1) * limit
		fo.SetSkip(int64(offset))
	} else {
		fo.SetNoCursorTimeout(true)
	}
	s.SetOpts(fo)
	return s.table.Where(s.filter).Count(), s.FetchAll(results)
}

// Run runs the given model.
func (s *Session) Run(size int32, callback func(*mongo.Cursor)) error {
	ctx := context.Background()

	fo := options.MergeFindOptions(s.findOpts...)
	fo.SetNoCursorTimeout(true)
	fo.SetBatchSize(size)
	s.SetOpts(fo)

	cur, err := s.collection.Find(ctx, s.filter, s.findOpts...)
	if err != nil {
		return err
	}

	defer cur.Close(ctx)
	for cur.Next(ctx) {
		callback(cur)
	}

	if err := cur.Err(); err != nil {
		return err
	}
	return nil
}
