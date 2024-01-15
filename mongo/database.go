package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Database mongo-driver database
type Database struct {
	database *mongo.Database
}

// CollectionNames returns the collection names present in database.
func (d *Database) CollectionNames() ([]string, error) {
	return d.database.ListCollectionNames(context.Background(), options.ListCollectionsOptions{})
}

// C returns collection.
func (d *Database) C(collection string) *Collection {
	return d.Collection(collection)
}

// Collection returns collection.
func (d *Database) Collection(collection string) *Collection {
	return &Collection{collection: d.database.Collection(collection)}
}
