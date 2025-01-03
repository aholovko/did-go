/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package verifier

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/trustbloc/did-go/doc/ld/processor"
	"github.com/trustbloc/did-go/doc/ld/proof"
	"github.com/trustbloc/did-go/doc/signature/api"
)

// keyResolver encapsulates key resolution.
type keyResolver interface {
	// Resolve will return public key bytes and the type of public key
	Resolve(id string) (*api.PublicKey, error)
}

// DocumentVerifier implements JSON LD document proof verification.
type DocumentVerifier struct {
	signatureSuites []api.VerifierSuite
	pkResolver      keyResolver
}

// New returns new instance of document verifier.
func New(resolver keyResolver, suites ...api.VerifierSuite) (*DocumentVerifier, error) {
	if len(suites) == 0 {
		return nil, errors.New("at least one suite must be provided")
	}

	return &DocumentVerifier{
		signatureSuites: suites,
		pkResolver:      resolver,
	}, nil
}

// Verify will verify document proofs.
// Deprecated. Please use vc-go/verifiable.VerifyDIDProof().
func (dv *DocumentVerifier) Verify(jsonLdDoc []byte, opts ...processor.Opts) error {
	var jsonLdObject map[string]interface{}

	err := json.Unmarshal(jsonLdDoc, &jsonLdObject)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json ld document: %w", err)
	}

	return dv.VerifyObject(jsonLdObject, opts...)
}

// VerifyObject will verify document proofs for JSON LD object.
// Deprecated. Please use vc-go/verifiable.VerifyDIDProof().
func (dv *DocumentVerifier) VerifyObject(jsonLdObject map[string]interface{}, opts ...processor.Opts) error {
	proofs, err := proof.GetProofs(jsonLdObject)
	if err != nil {
		return err
	}

	for _, p := range proofs {
		publicKeyID, err := p.PublicKeyID()
		if err != nil {
			return err
		}

		publicKey, err := dv.pkResolver.Resolve(publicKeyID)
		if err != nil {
			return err
		}

		suite, err := dv.getSignatureSuite(p.Type)
		if err != nil {
			return err
		}

		message, err := proof.CreateVerifyData(suite, jsonLdObject, p, opts...)
		if err != nil {
			return err
		}

		signature, err := getProofVerifyValue(p)
		if err != nil {
			return err
		}

		err = suite.Verify(publicKey, message, signature)
		if err != nil {
			return err
		}
	}

	return nil
}

// getSignatureSuite returns signature suite based on signature type.
func (dv *DocumentVerifier) getSignatureSuite(signatureType string) (api.VerifierSuite, error) {
	for _, s := range dv.signatureSuites {
		if s.Accept(signatureType) {
			return s, nil
		}
	}

	return nil, fmt.Errorf("signature type %s not supported", signatureType)
}

func getProofVerifyValue(p *proof.Proof) ([]byte, error) {
	switch p.SignatureRepresentation {
	case proof.SignatureProofValue:
		return p.ProofValue, nil
	case proof.SignatureJWS:
		return proof.GetJWTSignature(p.JWS)
	}

	return nil, fmt.Errorf("unsupported signature representation: %v", p.SignatureRepresentation)
}
