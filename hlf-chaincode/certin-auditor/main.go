/*
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	auditor "repo.antiersolutions.com/certin/certin-auditor/chaincode"
)

func main() {
	auditChaincode, err := contractapi.NewChaincode(&auditor.SmartContract{})
	if err != nil {
		log.Panicf("Error creating audit chaincode: %v", err.Error())
	}

	if err := auditChaincode.Start(); err != nil {
		log.Panicf("Error starting audit chaincode: %v", err.Error())
	}
}
