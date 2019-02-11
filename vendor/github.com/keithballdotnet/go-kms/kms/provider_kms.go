package kms

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/google/uuid"

	// "io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// encryptedKeyLength is the length of a 32 bit AES key encrypted using AES256-GCM
var encryptedKeyLength = 64

// KMSCryptoProvider is an implementation of encryption using a local storage
type KMSCryptoProvider struct {
	MasterKey []byte
}

// NewKMSCryptoProvider
func NewKMSCryptoProvider() (KMSCryptoProvider, error) {

	log.Println("Using KMS crypto provider...")

	// Ensure our config is ok...
	SetKMSCryptoConfig()

	log.Printf("KSM Crypto Path: %v", Config["GOKMS_KSMC_PATH"])

	// Check path
	_, err := os.Stat(Config["GOKMS_KSMC_PATH"])
	if err != nil {
		Exit(fmt.Sprintf("Can't use directory %s: %v", Config["GOKMS_KSMC_PATH"], err), 2)
	}

	var mkp MasterKeyProvider
	switch Config["GOKMS_CRYPTO_PROVIDER"] {
	case "gokms":
		mkp, err = NewGoKMSMasterKeyProvider()
	case "hsm":
		// Create crypto provider
		mkp, err = NewHSMMasterKeyProvider()
	default:
		mkp, err = NewGoKMSMasterKeyProvider()
	}

	key, err := mkp.GetKey()
	if err != nil {
		Exit(fmt.Sprintf("Can't get master key %s", err), 2)
	}

	return KMSCryptoProvider{MasterKey: key}, nil
}

// SetKMSCryptoConfig will check any required settings for this crypto-provider
func SetKMSCryptoConfig() {
	envFiles := []string{}

	providerConfig := map[string]string{
		"GOKMS_KSMC_PATH": filepath.Join(os.TempDir(), "go-kms", "keys"),
	}

	// Load all Environments variables
	for k, _ := range providerConfig {

		// Set default value...
		Config[k] = providerConfig[k]

		// Set env setting
		if os.Getenv(k) != "" {
			Config[k] = os.Getenv(k)
		}
	}
	// All variable MUST have a value but we can not verify the variable content
	for k, _ := range providerConfig {
		if Config[k] == "" {
			Exit(fmt.Sprintf("Problem with %s", k), 2)
		}
	}

	// Check file exists
	for _, v := range envFiles {
		_, err := os.Stat(Config[v])
		if err != nil {
			Exit(fmt.Sprintf("%s %s", v, err.Error()), 2)
		}
	}

	return
}

// EnableKey - will mark a key as enabled
func (cp KMSCryptoProvider) EnableKey(KeyID string) (KeyMetadata, error) {
	key, err := cp.GetKey(KeyID)
	if err != nil {
		return KeyMetadata{}, err
	}

	key.KeyMetadata.Enabled = true

	err = cp.SaveKey(key)

	if err != nil {
		return KeyMetadata{}, err
	}

	return key.KeyMetadata, nil
}

// DisableKey - will mark a key as disabled
func (cp KMSCryptoProvider) DisableKey(KeyID string) (KeyMetadata, error) {
	key, err := cp.GetKey(KeyID)
	if err != nil {
		return KeyMetadata{}, err
	}

	key.KeyMetadata.Enabled = false

	err = cp.SaveKey(key)

	if err != nil {
		return KeyMetadata{}, err
	}

	return key.KeyMetadata, nil
}

// ListKeys will list the available keys
func (cp KMSCryptoProvider) ListKeys() ([]KeyMetadata, error) {

	// Create slice of metadata to return
	metadata := make([]KeyMetadata, 0)

	files, _ := ioutil.ReadDir(Config["GOKMS_KSMC_PATH"])
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".key") {
			keyID := strings.TrimSuffix(f.Name(), ".key")
			key, err := cp.GetKey(keyID)
			if err != nil {
				log.Printf("ListKeys() got problem getting key %s: %v\n", keyID, err)
			} else {
				metadata = append(metadata, key.KeyMetadata)
			}
		}
	}

	return metadata, nil
}

// CreateKey will create a new key
func (cp KMSCryptoProvider) CreateKey(description string) (KeyMetadata, error) {

	// Create a new key id
	keyID := uuid.New().String()

	// Create metadata
	keyMetadata := KeyMetadata{
		KeyID:        keyID,
		Description:  description,
		CreationDate: time.Now().UTC(),
		Enabled:      true,
	}

	// Create a new secret key
	aesKey := cp.GenerateAesKey()

	// Create new key object
	key := Key{KeyMetadata: keyMetadata, AESKey: aesKey}

	// Persist the key to disk
	err := cp.SaveKey(key)
	if err != nil {
		return KeyMetadata{}, err
	}

	return keyMetadata, nil
}

// SaveKey will persist a key to disk
func (cp KMSCryptoProvider) SaveKey(key Key) error {
	// JSON -> byte
	keyData, err := json.Marshal(key)
	if err != nil {
		return err
	}

	// Create path to key
	keyPath := filepath.Join(Config["GOKMS_KSMC_PATH"], key.KeyMetadata.KeyID+".key")

	// Encrypt the key data with the user key and perist to disk..
	encryptedKey, err := AesGCMEncrypt(keyData, cp.MasterKey)
	if err != nil {
		return err
	}

	// Store key on disk
	err = ioutil.WriteFile(keyPath, encryptedKey, 0600)
	if err != nil {
		return err
	}

	return nil
}

// GetKey from the the store
func (cp KMSCryptoProvider) GetKey(KeyID string) (Key, error) {

	// Create path to key
	keyPath := filepath.Join(Config["GOKMS_KSMC_PATH"], KeyID+".key")

	// Read encrypted key from disk
	encryptedKey, err := ioutil.ReadFile(keyPath)
	if err != nil {
		log.Printf("GetKey() failed %s\n", err)
		return Key{}, err
	}

	// decrypt the data on disk with the users derived key
	decryptedData, err := AesGCMDecrypt(encryptedKey, cp.MasterKey)
	if err != nil {
		log.Printf("GetKey() failed %s\n", err)
		return Key{}, err
	}

	var key Key
	err = json.Unmarshal(decryptedData, &key)
	if err != nil {
		log.Printf("GetKey() failed %s\n", err)
		return Key{}, err
	}

	return key, nil
}

// ReEncrypt will decrypt with the current key, and rencrypt with the new key id
func (cp KMSCryptoProvider) ReEncrypt(data []byte, KeyID string) ([]byte, string, error) {

	// Decrypt the data
	plaintext, sourceKeyID, err := cp.Decrypt(data)
	if err != nil {
		return nil, "", err
	}

	// Encrypt with the new key
	ciphertext, err := cp.Encrypt(plaintext, KeyID)
	if err != nil {
		return nil, "", err
	}

	// return encrypted data
	return ciphertext, sourceKeyID, nil
}

// Encrypt will encrypt the data using the HSM
func (cp KMSCryptoProvider) Encrypt(data []byte, KeyID string) ([]byte, error) {

	key, err := cp.GetKey(KeyID)
	if err != nil {
		return nil, err
	}

	// Check to see if key is enabled
	if !key.KeyMetadata.Enabled {
		return nil, errors.New("Key is not enabled!")
	}

	encryptedData, err := AesGCMEncrypt(data, key.AESKey)
	if err != nil {
		return nil, err
	}

	// Encrypt the key ID used with the master key, so we can ID the key later on
	encryptedKey, err := AesGCMEncrypt([]byte(key.KeyMetadata.KeyID), cp.MasterKey)
	if err != nil {
		return nil, err
	}

	// Envelope the encrypted key with the encrypted data
	return append(encryptedKey, encryptedData...), nil
}

// Decrypt will decrypt the data using the HSM
func (cp KMSCryptoProvider) Decrypt(data []byte) ([]byte, string, error) {

	// Find the encrypted key ID
	encryptedKey := data[:encryptedKeyLength]
	encryptedData := data[encryptedKeyLength:]

	// Decrypt the key ID used in the encryption
	keyID, err := AesGCMDecrypt(encryptedKey, cp.MasterKey)
	if err != nil {
		return nil, "", err
	}

	// Get the key
	key, err := cp.GetKey(string(keyID))
	if err != nil {
		return nil, "", err
	}

	// Check to see if key is enabled
	if !key.KeyMetadata.Enabled {
		return nil, "", errors.New("Key is not enabled!")
	}

	// Let's decrypt the data
	decryptedData, err := AesGCMDecrypt(encryptedData, key.AESKey)
	if err != nil {
		log.Printf("Decrypt() failed %s\n", err)
		return nil, "", err
	}

	return decryptedData, string(keyID), nil
}

// Create a new Aes Secret
func (cp KMSCryptoProvider) GenerateAesKey() []byte {
	key := make([]byte, 32)
	io.ReadFull(rand.Reader, key)
	return key
}
