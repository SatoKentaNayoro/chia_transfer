/**
 _*_ @Author: IronHuang _*_
 _*_ @blog:https://www.dvpos.com/ _*_
 _*_ @Date: 2021/5/24 下午10:25 _*_
**/

package main

import (
	"bufio"
	"encoding/hex"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
)

func isEqualFile(srcPath, dstPath string) bool {
	srcStat, _ := os.Stat(srcPath)
	dstStat, _ := os.Stat(dstPath)
	if dstStat == nil || srcStat == nil {
		return false
	}
	if srcStat.Size() != dstStat.Size() {
		return false
	}

	srcCrc, _ := CalFileCrc(srcPath, srcStat.Size())
	dstCrc, _ := CalFileCrc(dstPath, dstStat.Size())
	if dstCrc != srcCrc {
		return false
	}
	return true
}

func CalFileCrc(filePath string, size int64) (string, error) {
	raw, err := MakeCalData(filePath, size)
	if err != nil {
		return "", err
	}
	return fileCrc32(raw)
}

func MakeCalData(filePath string, size int64) ([]byte, error) {
	const BUFFER_SIZE = 1024 * 4
	var sample []byte
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if size <= BUFFER_SIZE*30 {
		reader := bufio.NewReader(file)
		sample, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
	} else {
		buf := make([]byte, BUFFER_SIZE)
		chunk := size / 30
		for point := int64(0); point < size; point += chunk {
			file.Seek(point, 0)
			n, err := file.Read(buf)
			if err != nil && err != io.EOF {
				return nil, err
			}
			if n == 0 {
				break
			}
			// read the tail of file
			if point+BUFFER_SIZE < size && point+chunk >= size {
				bufTail := make([]byte, BUFFER_SIZE)
				if remain := size - (point + BUFFER_SIZE); remain < BUFFER_SIZE {
					bufTail = make([]byte, remain)
				}
				file.Seek(size-int64(len(bufTail)), 0)
				num, err := file.Read(bufTail)
				if err != nil && err != io.EOF {
					return nil, err
				}
				if num != 0 {
					buf = append(buf, bufTail...)
				}
			}

			sample = append(sample, buf...)
		}
	}
	return sample, nil
}

func fileCrc32(data []byte) (string, error) {
	_ieee := crc32.NewIEEE()
	_, err := _ieee.Write(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(_ieee.Sum([]byte(""))), nil
}
