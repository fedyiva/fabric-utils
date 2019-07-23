/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// This chaincode implements a simple map that is stored in the state.
// The following operations are available.

// Invoke operations
// put - requires two arguments, a key and value
// remove - requires a key
// get - requires one argument, a key, and returns a value
// keys - requires no arguments, returns all keys

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

// Init is a no-op
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	creator, err := stub.GetCreator()

	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get creator. Error: %s", err))
	}
	creatorString := url.PathEscape(string(creator[:]))
	fmt.Printf("Init creator: %s\n", creatorString)

	err = stub.PutState(creatorString, []byte("['read','write','admin']"))
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to put creator string. Error: %s", err))
	}

	state, err := stub.GetState(creatorString)

	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get creator state. Error: %s", err))
	}

	fmt.Printf("Check state: %s", string(state[:]))
	return shim.Success(nil)
}

func checkPermission(stub shim.ChaincodeStubInterface, permission string) bool {

	fmt.Printf("Checking permissions\n")
	creator, err := stub.GetCreator()
	if err != nil {
		fmt.Printf("Failed to get creator. Error: %s", err)
		return false
	}
	creatorString := url.PathEscape(string(creator[:]))
	fmt.Printf("Creator: %s\n", creatorString)

	state, err := stub.GetState(creatorString)
	fmt.Printf("State: %s", string(state[:]))
	return state != nil && strings.Contains(string(state[:]), permission)
}

// Invoke has two functions
// put - takes two arguments, a key and value, and stores them in the state
// remove - takes one argument, a key, and removes if from the state
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	fmt.Printf("Invoking\n")
	function, args := stub.GetFunctionAndParameters()
	fmt.Printf("Invoking chaincode with command %s\n", function)
	switch function {

	case "putPrivate":
		if !checkPermission(stub, "write") {
			fmt.Printf("Forbidden")
			return shim.Error(fmt.Sprintf("Forbidden"))
		}

		if len(args) < 2 {
			return shim.Error("put operation must include two arguments, a key and value")
		}
		key := args[0]
		value := args[1]

		// Check current value
		currentValue, err := stub.GetState(key)
		if err != nil {
			fmt.Printf("Error putting state %s", err)
			return shim.Error(fmt.Sprintf("put operation failed. Error updating state: %s", err))
		}
		fmt.Printf("Current value len: %d", len(currentValue))

		if err := stub.PutState(key, []byte(value)); err != nil {
			fmt.Printf("Error putting state %s", err)
			return shim.Error(fmt.Sprintf("put operation failed. Error updating state: %s", err))
		}

		return shim.Success(nil)

	case "getPrivate":
		if !checkPermission(stub, "read") {
			fmt.Printf("Forbidden")
			return shim.Error(fmt.Sprintf("Forbidden"))
		}

		if len(args) < 1 {
			return shim.Error("get operation must include one argument, a key")
		}
		key := args[0]

		value, err := stub.GetState(key)
		if err != nil {
			return shim.Error(fmt.Sprintf("get operation failed. Error accessing state: %s", err))
		}
		return shim.Success(value)

	case "permissionRequest":

		key := "permissionRequest"

		value, err := stub.GetState(key)
		if err != nil {
			return shim.Error(fmt.Sprintf("get permission request operation failed. Error accessing state: %s", err))
		}
		if value != nil {
			return shim.Error(fmt.Sprintf("permission request allready pending. Error accessing state: %s", err))
		}

		creator, err := stub.GetCreator()
		if err != nil {
			fmt.Printf("Failed to get creator. Error: %s", err)
			return shim.Error(fmt.Sprintf("Failed to get creator. Error: %s", err))
		}

		value = creator

		if err := stub.PutState(key, value); err != nil {
			fmt.Printf("Error putting state %s", err)
			return shim.Error(fmt.Sprintf("put operation failed. Error updating state: %s", err))
		}

		return shim.Success(nil)

	case "addReadWritePermission":
		if !checkPermission(stub, "admin") {
			fmt.Printf("Forbidden")
			return shim.Error(fmt.Sprintf("Forbidden"))
		}

		key := "permissionRequest"
		userCert, err := stub.GetState(key)
		if err != nil {
			return shim.Error(fmt.Sprintf("get permission request operation failed. Error accessing state: %s", err))
		}

		key = string(userCert[:])
		value := []byte("['read','write']")

		if err := stub.PutState(key, []byte(value)); err != nil {
			fmt.Printf("Error putting state %s", err)
			return shim.Error(fmt.Sprintf("put operation failed. Error updating state: %s", err))
		}

		message := new(Message)
		message.Payload = "ReadWritePermission"

		messageBytes, err := proto.Marshal(message)

		if err != nil {
			return shim.Error(fmt.Sprintf("Failed to create protobuf: %s", err))
		}

		if err := stub.SetEvent(key, messageBytes); err != nil {
			return shim.Error(fmt.Sprintf("put operation failed. Error emiting state update event with compositeKey: %s", err))
		}

		key = "lastGrantedUser"

		if err := stub.PutState(key, userCert); err != nil {
			fmt.Printf("Error putting state %s", err)
			return shim.Error(fmt.Sprintf("put operation failed. Error updating state: %s", err))
		}

		key = "permissionRequest"
		err = stub.DelState(key)
		if err != nil {
			return shim.Error(fmt.Sprintf("remove operation failed. Error updating state: %s", err))
		}

		return shim.Success(nil)

	case "addReadPermission":
		if !checkPermission(stub, "admin") {
			fmt.Printf("Forbidden")
			return shim.Error(fmt.Sprintf("Forbidden"))
		}

		key := "permissionRequest"
		userCert, err := stub.GetState(key)
		if err != nil {
			return shim.Error(fmt.Sprintf("get permission request operation failed. Error accessing state: %s", err))
		}

		key = string(userCert[:])
		value := []byte("['read']")

		if err := stub.PutState(key, value); err != nil {
			fmt.Printf("Error putting state %s", err)
			return shim.Error(fmt.Sprintf("put operation failed. Error updating state: %s", err))
		}

		message := new(Message)
		message.Payload = "ReadPermission"

		messageBytes, err := proto.Marshal(message)

		if err != nil {
			return shim.Error(fmt.Sprintf("Failed to create protobuf: %s", err))
		}

		if err := stub.SetEvent(key, messageBytes); err != nil {
			return shim.Error(fmt.Sprintf("put operation failed. Error emiting state update event with compositeKey: %s", err))
		}

		key = "lastGrantedUser"

		if err := stub.PutState(key, userCert); err != nil {
			fmt.Printf("Error putting state %s", err)
			return shim.Error(fmt.Sprintf("put operation failed. Error updating state: %s", err))
		}

		key = "permissionRequest"
		err = stub.DelState(key)
		if err != nil {
			return shim.Error(fmt.Sprintf("remove operation failed. Error updating state: %s", err))
		}

		return shim.Success(nil)

	default:
		return shim.Success([]byte("Unsupported operation"))
	}
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
