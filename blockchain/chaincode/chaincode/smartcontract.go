package chaincode

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
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
	PrometheusBaseURI := "http://192.168.0.181:3080/api/v1/"

	//TODO: check that either owner/consumer is executing this
	id := ctx.GetStub().GetTxID()

	//get current state of unpaid UsageSlice of from state
	currentSliceIterator, err := ctx.GetStub().GetStateByPartialCompositeKey("UsageSlice", []string{owner, consumer, "false"})
	if err != nil {
		return err
	}
	defer currentSliceIterator.Close()

	var currentSlice UsageSlice

	//if no slice with the key exists, create it else get from state
	if !currentSliceIterator.HasNext() {
		currentSlice = UsageSlice{
			ID:         id,
			Owner:      owner,
			Consumer:   consumer,
			K8sCPUMins: 0.00,
			K8sRAMMins: 0.00,
			Timestamp:  0,
			Paid:       false,
		}
	} else {
		//get slice from state
		currentSliceQueryResponse, err := currentSliceIterator.Next()
		if err != nil {
			return err
		}

		//make sure that there are no more unpaid slices, can remove this maybe
		if currentSliceIterator.HasNext() {
			return fmt.Errorf("Multiple UsageSlices with same key")
		}

		//parse reponse into object
		err = json.Unmarshal(currentSliceQueryResponse.Value, &currentSlice)
		if err != nil {
			return err
		}
	}

	// Get data from prometheus
	resp, err := http.Get(fmt.Sprintf("%squery_range?query=container_memory_usage_bytes&start=%d&end=1606229448.118&step=1&_=1606228308725",
		PrometheusBaseURI,
		currentSlice.Timestamp))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	prometheusResp, err := ioutil.ReadAll(resp.Body)
	return fmt.Errorf(string(prometheusResp))

	//TODO: compute usage difference

	//Add Values from previous state
	cpuMins := currentSlice.K8sCPUMins + 415.24
	ramMins := currentSlice.K8sRAMMins + 145.02
	timestamp := 635783233

	//check if there is any usage since the last update
	if cpuMins <= 0 && ramMins <= 0 {
		return fmt.Errorf("there is no resource to log since last sync")
	}

	slice := UsageSlice{
		ID:         id,
		Owner:      owner,
		Consumer:   consumer,
		K8sCPUMins: cpuMins,
		K8sRAMMins: ramMins,
		Timestamp:  timestamp,
		Paid:       false,
	}

	err = WriteUsageSliceToState(ctx, slice)
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
