/**
 _*_ @Author: IronHuang _*_
 _*_ @blog:https://www.dvpos.com/ _*_
 _*_ @Date: 2021/5/24 下午10:03 _*_
**/

package main

import (
	"errors"
	"fmt"
	"os"
)

func checkPathDoubledAndExisted(cfg *Config) error {
	doubledCheckMap := make(map[string]struct{})
	for _,mp := range cfg.MiddleTmps {
		err := checkPathExisted(mp.Path)
		if err != nil {
			return err
		}
		if _,ok := doubledCheckMap[mp.Path];!ok {
			doubledCheckMap[mp.Path] = struct{}{}
		}else {
			return errors.New(fmt.Sprintf("%s doubled,please check config file first",mp.Path))
		}
	}
	for _,fp := range cfg.FinalDirs {
		err := checkPathExisted(fp.Path)
		if err != nil {
			return err
		}
		if _,ok := doubledCheckMap[fp.Path];!ok {
			doubledCheckMap[fp.Path] = struct{}{}
		}else {
			return errors.New(fmt.Sprintf("%s doubled,please check config file first",fp.Path))
		}
	}
	return nil
}


func checkPathExisted(p string) error {
	_, err := os.Stat(p)
	return err
}