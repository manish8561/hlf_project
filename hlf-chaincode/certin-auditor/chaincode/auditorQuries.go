package auditor

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// QueryAudit allows all members of the channel to read a public audit
func (s *SmartContract) QueryAudit(ctx contractapi.TransactionContextInterface, auditID string) (*Audit, error) {
	log.Println("---------------1-------------------")

	log.Printf("ReadAudit: collection %v, ID %v", auditCollection, auditID)
	auditJSON, err := ctx.GetStub().GetPrivateData(auditCollection, auditID)
	//get the audit from chaincode state
	if err != nil {
		return nil, fmt.Errorf("failed to read audit: %v", err)
	}

	// No audit found, return empty response
	if auditJSON == nil {
		log.Printf("%v does not exist in collection %v", auditID, auditCollection)
		return nil, nil
	}

	var audit *Audit
	err = json.Unmarshal(auditJSON, &audit)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return audit, nil
}

// GetAuditByRange performs a range query based on the start and end keys provided. Range
// queries can be used to read data from private data collections, but can not be used in
// a transaction that also writes to private data.
func (s *SmartContract) GetAuditByRange(ctx contractapi.TransactionContextInterface, startKey string, endKey string) ([]*Audit, error) {

	resultsIterator, err := ctx.GetStub().GetPrivateDataByRange(auditCollection, startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	results := []*Audit{}

	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var audit *Audit
		err = json.Unmarshal(response.Value, &audit)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
		}

		results = append(results, audit)
	}

	return results, nil

}

// QueryAudits uses a query string to perform a query for assets.
// Query string matching state database syntax is passed in and executed as is.
// Supports ad hoc queries that can be defined at runtime by the client.
// If this is not desired, follow the QueryAuditByOwner example for parameterized queries.
// Only available on state databases that support rich query (e.g. CouchDB)
func (s *SmartContract) QueryAudits(ctx contractapi.TransactionContextInterface, queryString string) ([]*Audit, error) {

	resultsIterator, err := ctx.GetStub().GetPrivateDataQueryResult(auditCollection, queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	results := []*Audit{}

	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		var asset *Audit

		err = json.Unmarshal(response.Value, &asset)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
		}

		results = append(results, asset)
	}
	return results, nil
}
