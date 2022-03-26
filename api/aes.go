package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func main() {
	type VisitsData struct {
		// A    string `json:"a"`
		Ip   string `json:"ip"`
		Path string `json:"path"`
	}
	AES_KEY, _ := hex.DecodeString("26863727bf9e5378a657cd7f5177142f")
	text, _ := hex.DecodeString("f0619d1cdc428a9ad082b6fef6ce1c79d833c08174c27b22de9453eb3bdf3a34")
	iv, _ := hex.DecodeString("8534929688434090")

	dataJson := CBCDecrypter(text, AES_KEY, iv)

	var data VisitsData
	json.Unmarshal(dataJson, &data)
	fmt.Println(data.Path, data.Ip)
}

//解密函数
func CBCDecrypter(encrypter []byte, key []byte, iv []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println(err)
	}
	iv = IVPadding(iv)
	blockMode := cipher.NewCBCDecrypter(block, iv[:block.BlockSize()])
	result := make([]byte, len(encrypter))
	blockMode.CryptBlocks(result, encrypter)
	// 去除填充
	result = UnPKCS7Padding(result)
	return result
}

func UnPKCS7Padding(text []byte) []byte {
	// 取出填充的数据 以此来获得填充数据长度
	unPadding := int(text[len(text)-1])
	return text[:(len(text) - unPadding)]
}

//iv填充0
func IVPadding(sourceIV []byte) []byte {
	iv := [64]byte{}
	for i := 0; i < len(sourceIV); i++ {
		iv[i] = sourceIV[i]
	}
	return iv[:]
}
