// This package is all persistent data storage related code.
package db

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/pokt-network/pocket-core/config"
	"github.com/pokt-network/pocket-core/const"
	"github.com/pokt-network/pocket-core/logs"
	"github.com/pokt-network/pocket-core/node"
)

// "Add" 'puts' a node into the persistent data storage.
func (db *Database) Add(n node.Node) (*dynamodb.PutItemOutput, error) {
	db.Lock()
	defer db.Unlock()
	av, err := dynamodbattribute.MarshalMap(n)
	if err != nil {
		return nil, err
	}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(_const.DBTABLENAME),
	}
	res, err := db.dynamo.PutItem(input)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	return res, nil
}

// "Remove" 'deletes' a node from the persistent data storage.
func (db *Database) Remove(n node.Node) (*dynamodb.DeleteItemOutput, error) {
	db.Lock()
	defer db.Unlock()
	input := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"gid": {
				S: aws.String(n.GID),
			},
			"ip": {
				S: aws.String(n.IP),
			},
		},
		TableName: aws.String(_const.DBTABLENAME),
	}
	return db.dynamo.DeleteItem(input)
}

// "getAll" returns all nodes from the database.
func (db *Database) getAll() (*dynamodb.ScanOutput, error) {
	input := &dynamodb.ScanInput{TableName: aws.String(_const.DBTABLENAME)}
	return db.dynamo.Scan(input)
}

// "peersRefresh" updates the peerList and dispatchPeerList from the database every x time.
func peersRefresh() {
	var items []node.Node
	db := DB()
	for {
		db.Lock()
		output, err := DB().getAll()
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			db.Unlock()
			logs.NewLog(err.Error(), logs.PanicLevel, logs.JSONLogFormat)
		}
		// unmarshal the output from the database call
		err = dynamodbattribute.UnmarshalListOfMaps(output.Items, &items)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			db.Unlock()
			logs.NewLog(err.Error(), logs.PanicLevel, logs.JSONLogFormat)
		}
		pl := node.PeerList()
		pl.Set(items)
		pl.CopyToDP()
		db.Unlock()
		// every x minutes
		time.Sleep(_const.DBREFRESH * time.Minute)
	}
}

// "PeersRefresh" is a helper function that runs peersRefresh in a go routine
func PeersRefresh() {
	if config.GlobalConfig().Dispatch {
		go peersRefresh()
	}
}