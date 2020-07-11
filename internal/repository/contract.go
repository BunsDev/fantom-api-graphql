/*
Package repository implements repository for handling fast and efficient access to data required
by the resolvers of the API server.

Internally it utilizes RPC to access Opera/Lachesis full node for blockchain interaction. Mongo database
for fast, robust and scalable off-chain data storage, especially for aggregated and pre-calculated data mining
results. BigCache for in-memory object storage to speed up loading of frequently accessed entities.
*/
package repository

import (
	"bytes"
	"encoding/json"
	"fantom-api-graphql/internal/types"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"strings"
)

// Contract extract a smart contract information by account address, if available.
func (p *proxy) Contract(addr *common.Address) (*types.Contract, error) {
	return p.db.Contract(addr)
}

// Contracts returns list of smart contracts at Opera blockchain.
func (p *proxy) Contracts(validatedOnly bool, cursor *string, count int32) (*types.ContractList, error) {
	// go to the database for the list of contracts searched
	return p.db.Contracts(validatedOnly, cursor, count)
}

// cutCodeMetadata removes the IPFS/Swarm metadata information from the code
// for partial comparison. The current version of the Solidity compiler usually
// adds metadata to the end of the deployed byte code.
// @see https://solidity.readthedocs.io/en/latest/metadata.html
func cutCodeMetadata(bc []byte) []byte {
	// last 2 bytes are expected to contain metadata length
	bcLen := uint64(len(bc))
	cut := uint64(bc[bcLen-2])<<8 | uint64(bc[bcLen-1])

	// are we safely within the byte code size?
	if cut == 0 || cut >= bcLen-2 {
		return bc
	}

	return bc[:bcLen-cut-2]
}

// compareContractCode compares provided compiled code with the transaction input.
func compareContractCode(tx *types.Transaction, code string) (bool, error) {
	// decode the detail into byte array
	bc, err := hexutil.Decode(code)
	if err != nil {
		return false, err
	}

	// remove meta data hash from the byte code so we can compare raw
	// contract byte content. Such comparison is not perfect since
	// there could be changes in the source code not reflected
	// in the byte code. (variables renamed, unused code introduced, etc.)
	// Safer would be to use full CBOR parser here.
	bc = cutCodeMetadata(bc)

	// Is the transaction input shorter than the compiled contract?
	// If so there is no chance for pass.
	if len(tx.InputData) < len(bc) {
		return false, nil
	}

	// compare only up to <bc> length, the rest is metadata
	// and constructor parameters
	res := bytes.Compare(bc, tx.InputData[:len(bc)])

	// return the comparison result
	return res == 0, nil
}

// updateContractDetails updates local contract details from the provided compiler
// output.
func updateContractDetails(sc *types.Contract, detail *compiler.Contract) {
	// copy compiler information
	var str strings.Builder
	str.WriteString(detail.Info.Language)
	str.WriteString(" ")
	str.WriteString(detail.Info.LanguageVersion)
	sc.Compiler = str.String()

	// copy ABI
	abi, err := json.Marshal(detail.Info.AbiDefinition)
	if err == nil {
		sc.Abi = string(abi)
	}

	// copy the source code
	sc.SourceCode = detail.Info.Source
}

// ValidateContract tries to validate contract byte code using
// provided source code. If successful, the contract information
// is updated the the repository.
func (p *proxy) ValidateContract(sc *types.Contract) error {
	// get the byte code of the actual contract
	tx, err := p.Transaction(&sc.TransactionHash)
	if err != nil {
		p.log.Errorf("can not get contract deployment transaction; %s", err.Error())
		return err
	}

	// try to compile the source code provided
	contracts, err := compiler.CompileSolidityString(p.solCompiler, sc.SourceCode)
	if err != nil {
		p.log.Errorf("solidity code compilation failed")
		return err
	}

	return p.validateContractSet(contracts, tx, sc)
}

// ValidateContract tries to validate contract byte code using
// provided source code. If successful, the contract information
// is updated the the repository.
func (p *proxy) validateContractSet(contracts map[string]*compiler.Contract, tx *types.Transaction, sc *types.Contract) error {
	// loop over contracts ad try to validate one of them
	for name, detail := range contracts {
		// check if the compiled byte code match with the deployed contract
		match, err := compareContractCode(tx, detail.Code)
		if err != nil {
			p.log.Errorf("contract byte code comparison failed")
			return err
		}

		// we have the winner
		if match {
			// set the contract name if not done already
			if 0 == len(sc.Name) {
				sc.Name = strings.TrimPrefix(name, "<stdin>:")
			}

			// update the contract data
			updateContractDetails(sc, detail)

			// write update to the database
			if err := p.db.UpdateContract(sc); err != nil {
				p.log.Errorf("contract validation failed due to db error; %s", err.Error())
				return err
			}

			// inform about success
			p.log.Debugf("contract %s [%s] validated", sc.Address.String(), name)

			// inform the upper instance we have a winner
			return nil
		}
	}

	// validation fails
	return fmt.Errorf("contract source code does not match with the deployed byte code")
}
