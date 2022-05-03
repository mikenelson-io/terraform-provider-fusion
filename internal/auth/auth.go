/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/oauth2"
)

const DefaultAuthNEndpoint = "https://api.pure1.purestorage.com/oauth2/1.0/token"
const AuthNEndpointOverrideEnvVarName = "PURE1_AUTHENTICATION_ENDPOINT"

// Connects to Pure1 Authentication endpoint with issuerID signed with private key specified by given path
// This returns an access token that is good for one hour, in any exceptional cases it returns an empty string
func GetPure1SelfSignedAccessTokenGoodForOneHour(ctx context.Context, issuerId string, privateKeyPath string) (string, error) {
	authNEndpoint := os.Getenv(AuthNEndpointOverrideEnvVarName)
	if authNEndpoint == "" {
		authNEndpoint = DefaultAuthNEndpoint
	}

	privateKeyData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key file path:%s err:%w", privateKeyPath, err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key path:%s err:%w", privateKeyPath, err)
	}

	signedIdentityToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.StandardClaims{
		Issuer:    issuerId,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(3600 * time.Second).Unix(),
	}).SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign identity token path:%s err:%w", privateKeyPath, err)
	}

	config := oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: authNEndpoint}}
	exchangedToken, err := config.Exchange(ctx, "",
		oauth2.SetAuthURLParam("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange"),
		oauth2.SetAuthURLParam("subject_token", signedIdentityToken),
		oauth2.SetAuthURLParam("subject_token_type", "urn:ietf:params:oauth:token-type:jwt"),
	)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token endpoint:%s path:%s err:%w", authNEndpoint, privateKeyPath, err)
	}
	return exchangedToken.AccessToken, nil
}
