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
	"os/signal"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

var log = logging.Logger("transfer")
var stop = false

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
	stopSignal := make(chan os.Signal, 2)
	signal.Notify(stopSignal, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case <-stopSignal:
			stop = true
		}
	}()
	for {
		if stop {
			log.Warn("stop by signal,waiting all working task stop")
			for {
				allStop := true
				for _, f := range finalDirWithLockList {
					if f.Status != StatusOnFree {
						allStop = false
					}
				}
				if allStop {
					log.Warn("all working task stopped")
					return
				}
				time.Sleep(time.Second * 5)
			}
		}
		// add middle Files
		for _, mp := range config.MiddleTmps {
			err := filepath.Walk(mp, func(path string, info os.FileInfo, err error) error {
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
				return err
			})
			if err != nil {
				log.Error(err)
				return
			}
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
							log.Infof("start myCopy %s", srcFile.FilePath)
							fdl.Flock.Lock()
							fdl.Status = StatusOnWorking
							fdl.Flock.Unlock()
							err2 := myCopy(srcFile.FilePath, temporaryPath)
							if err2 != nil {
								log.Errorf("myCopy %s error: %s", srcFile.FilePath, err.Error())
								os.Remove(temporaryPath)
							}
							if !isEqualFile(srcFile.FilePath, temporaryPath) {
								log.Errorf("myCopy %s error: dst not equal with src", srcFile.FilePath)
								os.Remove(temporaryPath)
							}
							fdl.Flock.Lock()
							fdl.Status = StatusOnFree
							fdl.Flock.Unlock()
							// del src file
							os.Remove(srcFile.FilePath)
							log.Infof("myCopy %s to %s done", srcFile.FilePath, temporaryPath)
							<-threadChan
						}()
					}
				}
			}
		default:
			time.Sleep(time.Second * 5)
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

func myCopy(src, dst string) (err error) {
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
		if stop {
			return errors.New("StoppedBySyscall")
		}

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
