#!/bin/bash
sudo ./network.sh down
sudo ./network.sh up createChannel -ca
sudo ./network.sh deployCC -ccn sinode -ccp ../chaincode -ccl go -ccep "AND('Org1MSP.peer','Org2MSP.peer')"
cd ../test-app
sudo rm -rf wallet
sudo rm -rf keystore
sudo rm assetTransfer
go build assetTransfer.go
sudo ./assetTransfer