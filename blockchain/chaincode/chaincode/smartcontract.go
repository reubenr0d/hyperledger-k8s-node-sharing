package chaincode

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

// Asset describes basic details of what makes up a simple asset
type Asset struct {
	ID             string `json:"ID"`
	Color          string `json:"color"`
	Size           int    `json:"size"`
	Owner          string `json:"owner"`
	AppraisedValue int    `json:"appraisedValue"`
}

// UsageSlice describes a K8s usage block that
type UsageSlice struct {
	ID         string  `json:"id"`
	Owner      string  `json:"owner_id"`
	Consumer   string  `json:"consumer_id"`
	K8sCPUMins float64 `json:"k8s_cpu_mins"`
	K8sRAMMins float64 `json:"k8s_ram_mins"`
	Timestamp  int     `json:"epoch"`
	Paid       bool    `json:"is_paid"`
}

// TransferUsageSlice checks for new usage and transfers slice from owner to consumer
func (s *SmartContract) TransferUsageSlice(ctx contractapi.TransactionContextInterface, owner string, consumer string) error {
	//TODO: check that either owner/consumer is executing this

	//TODO: get current state of unpaid UsageSlice of from state

	//TODO: get new ram usage difference
	cpuMins := 415.24
	ramMins := 145.02
	timestamp := 635783233

	//check if there is any usage since the last update
	if cpuMins <= 0 && ramMins <= 0 {
		return fmt.Errorf("there is no resource to log since last sync")
	}

	id := ctx.GetStub().GetTxID()
	slice := UsageSlice{
		ID:         id,
		Owner:      owner,
		Consumer:   consumer,
		K8sCPUMins: cpuMins,
		K8sRAMMins: ramMins,
		Timestamp:  timestamp,
		Paid:       false,
	}

	err := WriteUsageSliceToState(ctx, slice)
	if err != nil {
		return err
	}
	return nil
}

// WriteUsageSliceToState writes a UsageSlice to the state
func WriteUsageSliceToState(ctx contractapi.TransactionContextInterface, slice UsageSlice) error {
	sliceJSON, err := json.Marshal(slice)
	if err != nil {
		return err
	}

	//Create composite ID of owner, consumer, timestamp so that it easy while getting state
	sliceCompositeKey, _ := ctx.GetStub().CreateCompositeKey("UsageSlice", []string{slice.Owner, slice.Consumer, strconv.FormatBool(slice.Paid)}) // forammted bool to string as only accept strings in arrray

	// add to state
	err = ctx.GetStub().PutState(sliceCompositeKey, sliceJSON)
	if err != nil {
		return fmt.Errorf("failed to put to world state. %v", err)
	}
	return nil
}

// CreateAsset issues a new asset to the world state with given details.
func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, id string, color string, size int, owner string, appraisedValue int) error {
	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the asset %s already exists", id)
	}

	asset := Asset{
		ID:             id,
		Color:          color,
		Size:           size,
		Owner:          owner,
		AppraisedValue: appraisedValue,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// ReadAsset returns the asset stored in the world state with given id.
func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, id string) (*Asset, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("the asset %s does not exist", id)
	}

	var asset Asset
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// UpdateAsset updates an existing asset in the world state with provided parameters.
func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, id string, color string, size int, owner string, appraisedValue int) error {
	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the asset %s does not exist", id)
	}

	// overwriting original asset with new asset
	asset := Asset{
		ID:             id,
		Color:          color,
		Size:           size,
		Owner:          owner,
		AppraisedValue: appraisedValue,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// DeleteAsset deletes an given asset from the world state.
func (s *SmartContract) DeleteAsset(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the asset %s does not exist", id)
	}

	return ctx.GetStub().DelState(id)
}

// AssetExists returns true when asset with given ID exists in world state
func (s *SmartContract) AssetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return assetJSON != nil, nil
}

// TransferAsset updates the owner field of asset with given id in world state.
func (s *SmartContract) TransferAsset(ctx contractapi.TransactionContextInterface, id string, newOwner string) error {
	asset, err := s.ReadAsset(ctx, id)
	if err != nil {
		return err
	}

	asset.Owner = newOwner
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// GetAllAssets returns all assets found in world state
func (s *SmartContract) GetAllAssets(ctx contractapi.TransactionContextInterface) ([]*UsageSlice, error) {
	// open-ended query of all UsageSlices in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByPartialCompositeKey("UsageSlice", []string{})
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var assets []*UsageSlice
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset UsageSlice
		err = json.Unmarshal(queryResponse.Value, &asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}
