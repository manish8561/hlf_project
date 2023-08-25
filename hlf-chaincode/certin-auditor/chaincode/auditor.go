package auditor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const auditCollection = "auditCollection"

// SmartContract provides functions for managing an Audit
type SmartContract struct {
	contractapi.Contract
}

// Audit describes basic details of what makes up a simple audit
type Audit struct {
	ID         string    `json:"ID"`
	Name       string    `json:"name"`
	AuditType  string    `json:"auditType"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Reason     string    `json:"reason"`
	Status     string    `json:"status"`
	Auditor    string    `json:"auditor"`
	Orgs       []string  `json:"organizations"`
	ReportFile string    `json:"reportFile"`
	Tenure     uint      `json:"tenure"` //in days
}

/**
 * inital function to state value
 */
func (s *SmartContract) Initialize(ctx contractapi.TransactionContextInterface, initValue string) error {
	log.Println("Init invoked")
	// var Aval, Bval int // Audit holdings
	var err error

	if len(initValue) <= 0 {
		return fmt.Errorf("Initial value will not be a non-empty string")
	}

	// Initialize the chaincode
	A := "ID"

	// Write the state to the ledger
	err = ctx.GetStub().PutState(A, []byte(initValue))
	if err != nil {
		return err
	}

	log.Println("Init returning with success")
	return nil
}

/**
 * get state public value
 */
func (s *SmartContract) GetContractID(ctx contractapi.TransactionContextInterface) (string, error) {
	A := "ID"

	value, err := ctx.GetStub().GetState(A)
	if err != nil {
		return "", fmt.Errorf("failed to get audit object %v: %v", A, err)
	}

	return string(value), nil
}

// CreateAudit creates on audit on the public channel. The identity that
// submits the transacion becomes the auditor of the audit
func (s *SmartContract) CreateAudit(ctx contractapi.TransactionContextInterface) error {

	// get ID of submitting client
	clientID, err := submittingClientIdentity(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	// get org of submitting client
	clientOrgID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	// Get new audit from transient map
	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("error getting transient: %v", err)
	}

	// audit properties are private, therefore they get passed in transient field, instead of func args
	transientAuditJSON, ok := transientMap["audit_properties"]
	if !ok {
		//log error to stdout
		return fmt.Errorf("audit properties not found in the transient map input")
	}

	type auditTransientInput struct {
		ID        string `json:"auditID"`
		Name      string `json:"name"`
		AuditType string `json:"auditType"`
		Reason    string `json:"reason"`
		Tenure    uint   `json:"tenure"`
	}
	var auditInput auditTransientInput
	err = json.Unmarshal(transientAuditJSON, &auditInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if len(auditInput.AuditType) == 0 {
		return fmt.Errorf("Audit Type field must be a non-empty string")
	}
	if len(auditInput.Name) == 0 {
		return fmt.Errorf("Name field must be a non-empty string")
	}
	if len(auditInput.ID) == 0 {
		return fmt.Errorf("auditID field must be a non-empty string")
	}
	if len(auditInput.Reason) == 0 {
		return fmt.Errorf("Reason field must be a non-empty string")
	}
	if auditInput.Tenure <= 0 {
		return fmt.Errorf("Tenure field must be a positive integer")
	}

	// creating the audit
	audit := Audit{
		ID:         auditInput.ID,
		Name:       auditInput.Name,
		AuditType:  auditInput.AuditType,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Reason:     auditInput.Reason,
		Tenure:     auditInput.Tenure,
		Status:     "pending",
		ReportFile: "",
		Auditor:    clientID,
		Orgs:       []string{clientOrgID},
	}

	// Verify that the client is submitting request to peer in their organization
	// This is to ensure that a client from another org doesn't attempt to read or
	// write private data from this peer.
	// err = verifyClientOrgMatchesPeerOrg(ctx)
	// if err != nil {
	// 	return fmt.Errorf("CreateAudit cannot be performed: Error %v", err)
	// }
	log.Println("---------------4-------------------")

	// Check if audit already exists
	auditAsBytes, err := ctx.GetStub().GetPrivateData(auditCollection, audit.ID)
	if err != nil {
		return fmt.Errorf("failed to get audit: %v", err)
	} else if auditAsBytes != nil {
		fmt.Println("Audit already exists with same id: " + audit.ID)
		return fmt.Errorf("this audit already exists: " + audit.ID)
	}

	auditJSONBytes, err := json.Marshal(audit)
	if err != nil {
		return err
	}
	// Save audit to private data collection
	// Typical logger, logs to stdout/file in the fabric managed docker container, running this chaincode
	// Look for container name like dev-peer0.org1.example.com-{chaincodename_version}-xyz
	log.Printf("CreateAudit Put: collection %v, ID %v, owner %v", auditCollection, audit.ID, clientID)

	err = ctx.GetStub().PutPrivateData(auditCollection, audit.ID, auditJSONBytes)
	if err != nil {
		return fmt.Errorf("failed to put audit into private data collecton: %v", err)
	}
	log.Println("---------------6-------------------")

	return nil
}

// PurgeAudit can be used by the auditor of the audit to delete the audit
// Trigger removal of the audit
func (s *SmartContract) PurgeAudit(ctx contractapi.TransactionContextInterface) error {

	transientMap, err := ctx.GetStub().GetTransient()
	if err != nil {
		return fmt.Errorf("Error getting transient: %v", err)
	}

	// Audit properties are private, therefore they get passed in transient field
	transientDeleteJSON, ok := transientMap["audit_purge"]
	if !ok {
		return fmt.Errorf("audit to purge not found in the transient map")
	}

	type auditPurge struct {
		ID string `json:"auditID"`
	}

	var auditPurgeInput auditPurge
	err = json.Unmarshal(transientDeleteJSON, &auditPurgeInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	if len(auditPurgeInput.ID) == 0 {
		return fmt.Errorf("auditID field must be a non-empty string")
	}

	// Verify that the client is submitting request to peer in their organization
	err = verifyClientOrgMatchesPeerOrg(ctx)
	if err != nil {
		return fmt.Errorf("PurgeAudit cannot be performed: Error %v", err)
	}

	log.Printf("Purging Audit: %v", auditPurgeInput.ID)

	// Note that there is no check here to see if the id exist; it might have been 'deleted' already
	// so a check here is pointless. We would need to call purge irrespective of the result
	// A delete can be called before purge, but is not essential
	// delete the audit from state
	err = ctx.GetStub().PurgePrivateData(auditCollection, auditPurgeInput.ID)
	if err != nil {
		return fmt.Errorf("failed to purge state from audit collection: %v", err)
	}

	return nil

}

// verifyClientOrgMatchesPeerOrg is an internal function used verify client org id and matches peer org id.
func verifyClientOrgMatchesPeerOrg(ctx contractapi.TransactionContextInterface) error {
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed getting the client's MSPID: %v", err)
	}
	peerMSPID, err := shim.GetMSPID()
	if err != nil {
		return fmt.Errorf("failed getting the peer's MSPID: %v", err)
	}

	if clientMSPID != peerMSPID {
		return fmt.Errorf("client from org %v is not authorized to read or write private data from an org %v peer", clientMSPID, peerMSPID)
	}

	return nil
}
/**
 * submit id of the client id
 */
func submittingClientIdentity(ctx contractapi.TransactionContextInterface) (string, error) {
	b64ID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("Failed to read clientID: %v", err)
	}
	decodeID, err := base64.StdEncoding.DecodeString(b64ID)
	if err != nil {
		return "", fmt.Errorf("failed to base64 decode clientID: %v", err)
	}
	return string(decodeID), nil
}
