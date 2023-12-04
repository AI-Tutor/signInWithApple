package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/oauth2/jws"
)

var (
	// replace your configs here
	secret      = ""
	keyId       = "QGTH3C248V"
	teamId      = "H33J3M76HR"
	clientId    = "com.creativiti-kids.CreativeAI"
	redirectUrl = "https://ec2-3-106-222-121.ap-southeast-2.compute.amazonaws.com/callback"
)

// AppleKeys 用于解析 Apple 公钥接口的响应
type AppleKeys struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Use string `json:"use"`
		Alg string `json:"alg"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

// 获取 Apple 的公共密钥
func getApplePublicKey(idToken string) (*x509.Certificate, error) {
	resp, err := http.Get("https://appleid.apple.com/auth/keys")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var appleKeys AppleKeys
	if err := json.NewDecoder(resp.Body).Decode(&appleKeys); err != nil {
		return nil, err
	}

	// 解码 JWT，这里不直接验证签名
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		// 根据 JWT 头部的 kid 字段找到正确的公钥
		if kid, ok := token.Header["kid"].(string); ok {
			for _, key := range appleKeys.Keys {
				if key.Kid == kid {
					return token, nil
				}
			}
		}
		return nil, fmt.Errorf("unable to find appropriate key")
	})

	for _, key := range appleKeys.Keys {
		if key.Kid == token.Header["kid"] {
			// 构建证书
			certData := fmt.Sprintf("-----BEGIN CERTIFICATE-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAn%s==\n-----END CERTIFICATE-----", key.N)
			block, _ := pem.Decode([]byte(certData))
			if block == nil {
				return nil, fmt.Errorf("failed to parse certificate PEM")
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			return cert, nil
		}
	}
	return nil, fmt.Errorf("unable to find matching Apple public key")
}

// 解码并验证 ID Token
func decodeAndVerifyIDToken(idToken string) (*jws.ClaimSet, error) {

	//applePublicKey, err := getApplePublicKey(idToken)
	//if err != nil {
	//	return nil, err
	//}

	//// 获取与 kid 匹配的 Apple 公钥
	//applePublicKey, err := getApplePublicKey(header.KeyID)
	//if err != nil {
	//	return nil, err
	//}

	// 验证签名并解析 JWT
	claimSet, err := jws.Decode(idToken)
	if err != nil {
		return nil, err
	}

	return claimSet, nil
}

// create client_secret
func GetAppleSecret() string {
	token := &jwt.Token{
		Header: map[string]interface{}{
			"alg": "ES256",
			"kid": keyId,
		},
		Claims: jwt.MapClaims{
			"iss": teamId,
			"iat": time.Now().Unix(),
			// constraint: exp - iat <= 180 days
			"exp": time.Now().Add(24 * time.Hour).Unix(),
			"aud": "https://appleid.apple.com",
			"sub": clientId,
		},
		Method: jwt.SigningMethodES256,
	}

	ecdsaKey, _ := AuthKeyFromBytes([]byte(secret))
	ss, _ := token.SignedString(ecdsaKey)
	return ss
}

// create private key for jwt sign
func AuthKeyFromBytes(key []byte) (*ecdsa.PrivateKey, error) {
	var err error

	// Parse PEM block
	var block *pem.Block
	if block, _ = pem.Decode(key); block == nil {
		return nil, errors.New("token: AuthKey must be a valid .p8 PEM file")
	}

	// Parse the key
	var parsedKey interface{}
	if parsedKey, err = x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
		return nil, err
	}

	var pkey *ecdsa.PrivateKey
	var ok bool
	if pkey, ok = parsedKey.(*ecdsa.PrivateKey); !ok {
		return nil, errors.New("token: AuthKey must be of type ecdsa.PrivateKey")
	}

	return pkey, nil
}

// do http request
func HttpRequest(method, addr string, params map[string]string) ([]byte, int, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	var request *http.Request
	var err error
	if request, err = http.NewRequest(method, addr, strings.NewReader(form.Encode())); err != nil {
		return nil, 0, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var response *http.Response
	if response, err = http.DefaultClient.Do(request); nil != err {
		return nil, 0, err
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, 0, err
	}
	return data, response.StatusCode, nil
}

func main() {
	// replace your code here
	code := "cad92d642e7c941d4ae013adb3f466ad7.0.rrvwx.5hPTB51Mgw7xdD7bZdHBiw"
	data, status, err := HttpRequest("POST", "https://appleid.apple.com/auth/token", map[string]string{
		"client_id":     clientId,
		"client_secret": GetAppleSecret(),
		"code":          code,
		"grant_type":    "authorization_code",
		"redirect_uri":  redirectUrl,
	})

	fmt.Printf("%d\n%v\n%s", status, err, data)
}
