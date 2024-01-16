package mongo

import (
	"fmt"
	"log"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const Table = "persons"

type Person struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

var session *Session

func init() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	session = New(`mongodb://root:1234@test.localhost:27017/easygram?authsource=admin&connectTimeoutMS=2000&replicaSet=rs0&readPreference=primaryPreferred&maxStalenessSeconds=120`)

	_, err := session.C(Table).Index(
		bson.M{
			"unique": true,
			"keys": bson.D{
				{"name", 1},
			},
		},
		bson.M{
			"keys": bson.D{
				{"created_at", 1},
			},
		},
	)
	if err != nil {
		log.Println(err)
	}
}

func TestPagination(t *testing.T) {
	var err error
	var total int64
	var results []Person
	if total, err = session.C(Table).Where(bson.D{}).SetOpts(options.Find().SetSort(bson.D{{"_id", -1}})).Pagination(2, 2, &results); err != nil {
		log.Println(err)
	}

	log.Printf("total: %v", total)
	for _, r := range results {
		log.Println(r.Name)
	}
}

func TestMain(t *testing.T) {

	// Find find all
	var result []Person
	if err := session.C(Table).Where(bson.D{}).FetchAll(&result); err != nil {
		log.Println(err)
	}

	for _, r := range result {
		log.Println(r.Name)
	}

	// Update one
	if _, err := session.C(Table).Where(bson.D{{"name", "name1"}}).UpdateOne(bson.M{"$set": bson.M{"name": "name01"}}); err != nil {
		log.Println(err)
	}

	// Update update all
	info, err := session.C(Table).Where(bson.D{{"name", "name01"}}).Update(bson.M{"$set": bson.M{"name": "name"}})
	if err != nil {
		log.Println(err)
	}
	log.Printf("%+v", info)

	// Remove one
	if err := session.C(Table).Where(bson.D{{"name", "name"}}).RemoveOne(); err != nil {
		log.Println(err)
	}

	// RemoveAll
	if err := session.C(Table).Where(bson.D{{"name", "name"}}).Remove(); err != nil {
		log.Println(err)
	}

	// Insert
	if _, err := session.C(Table).Insert(bson.M{"name": "name"}); err != nil {
		log.Println(err)
	}

	// InsertAll
	var docs []interface{}
	for index := 0; index < 10; index++ {
		docs = append(docs, bson.M{
			"name":       fmt.Sprintf("name%d", index),
			"created_at": time.Now(),
		})
	}

	if _, err := session.C(Table).InsertAll(docs); err != nil {
		log.Println(err)
	}

	// Count
	count := session.C(Table).Where(bson.D{{"name", "name"}}).Count()
	log.Println(count)
}
