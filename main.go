/**
 _*_ @Author: IronHuang _*_
 _*_ @blog:https://www.dvpos.com/ _*_
 _*_ @Date: 2021/5/24 下午9:46 _*_
**/

package main

import (
	"errors"
	"fmt"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	logging "github.com/ipfs/go-log"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

var log = logging.Logger("transfer")

const (
	StatusOnWorking = "StatusOnWorking"
	StatusOnFree    = "StatusOnFree"
)

type Config struct {
	MiddleTmps []string
	FinalDirs  []string
}

type FinalDirWithLock struct {
	Path   string
	Status string
	Flock  *sync.Mutex
}

type SrcFile struct {
	SrcDir   string
	FilePath string
}

func main() {
	lotuslog.SetupLogLevels()
	config, err := loadConfig()
	if err != nil {
		log.Error(err)
		return
	}
	// init finalDir list
	finalDirWithLockList := initFinalDirs(config)

	var threadChan = make(chan struct{}, len(finalDirWithLockList))
	var toDoFiles = make(map[string]SrcFile)
	for {
		// add middle Files
		for _, mp := range config.MiddleTmps {
			filepath.Walk(mp, func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() && info.Mode().IsRegular() {
					fullPath := mp + "/" + info.Name()
					srcFile := SrcFile{
						SrcDir:   mp,
						FilePath: fullPath,
					}
					// if no in, add one
					if _, ok := toDoFiles[fullPath]; !ok {
						toDoFiles[fullPath] = srcFile
					}
				}
				return nil
			})
		}
		select {
		case threadChan <- struct{}{}:
			for _, v := range toDoFiles {
				srcFile := v
				for _, f := range finalDirWithLockList {
					fdl := f
					temporaryPath := strings.Replace(srcFile.FilePath, srcFile.SrcDir, fdl.Path, 1)
					_, err := os.Stat(temporaryPath)
					if err == nil {
						//delete(toDoFiles, k)
						break
					}
					srcStat, err := os.Stat(srcFile.FilePath)
					if err != nil {
						log.Error(err)
						break
					}
					var dstStat = new(syscall.Statfs_t)
					_ = syscall.Statfs(srcFile.SrcDir, dstStat)

					if fdl.Status == StatusOnFree && dstStat.Bavail*uint64(dstStat.Bsize) >= uint64(srcStat.Size()) {
						go func() {
							fdl.Flock.Lock()
							fdl.Status = StatusOnWorking
							fdl.Flock.Unlock()
							err2 := copy(srcFile.FilePath, temporaryPath)
							if err2 != nil || !isEqualFile(srcFile.FilePath, temporaryPath) {
								os.Remove(temporaryPath)
							}
							fdl.Flock.Lock()
							fdl.Status = StatusOnFree
							fdl.Flock.Unlock()
							// del src file
							os.Remove(srcFile.FilePath)
							log.Infof("copy %s to %s done", srcFile.FilePath, temporaryPath)
							<-threadChan
						}()
					}
				}
			}
		default:
			time.Sleep(time.Minute * 10)
		}
	}
}

func loadConfig() (*Config, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}
	raw, err := ioutil.ReadFile(path.Join(currentUser.HomeDir, "chia_transfer.yaml"))
	if err != nil {
		return nil, err
	}
	config := Config{}
	err = yaml.Unmarshal(raw, &config)
	if err != nil {
		return nil, err
	}
	log.Info(config)
	if len(config.FinalDirs) <= 0 || len(config.MiddleTmps) <= 0 {
		return nil, errors.New("len dirs error")
	}
	for i, p := range config.MiddleTmps {
		pp := strings.TrimRight(p, "/")
		config.MiddleTmps[i] = pp
	}

	for idx, fp := range config.FinalDirs {
		pp := strings.TrimRight(fp, "/")
		config.MiddleTmps[idx] = pp
	}

	err = checkPathDoubledAndExisted(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func initFinalDirs(cfg *Config) []*FinalDirWithLock {
	var fdlList = make([]*FinalDirWithLock, 0)
	for _, v := range cfg.FinalDirs {
		fdl := new(FinalDirWithLock)
		fdl.Path = v
		fdl.Status = StatusOnFree
		fdlList = append(fdlList, fdl)
	}
	return fdlList
}

func copy(src, dst string) (err error) {
	const BufferSize = 1 * 1024 * 1024
	buf := make([]byte, BufferSize)

	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		err2 := source.Close()
		if err2 != nil && err == nil {
			err = err2
		}
	}()

	if err != nil {
		return err
	}
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		err2 := destination.Close()
		if err2 != nil && err == nil {
			err = err2
		}
	}()

	for {
		//if stop {
		//	return errors.New(move_common.StoppedBySyscall)
		//}

		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		// 限速
		//if singleThreadMBPS != 0 {
		//	sleepTime := 1000000 / int64(singleThreadMBPS)
		//	time.Sleep(time.Microsecond * time.Duration(sleepTime))
		//}

		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}
	return
}
