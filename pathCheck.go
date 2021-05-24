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
	for _, mp := range cfg.MiddleTmps {
		err := checkPathExisted(mp)
		if err != nil {
			return err
		}
		if _, ok := doubledCheckMap[mp]; !ok {
			doubledCheckMap[mp] = struct{}{}
		} else {
			return errors.New(fmt.Sprintf("%s doubled,please check config file first", mp))
		}
	}
	for _, fp := range cfg.FinalDirs {
		err := checkPathExisted(fp)
		if err != nil {
			return err
		}
		if _, ok := doubledCheckMap[fp]; !ok {
			doubledCheckMap[fp] = struct{}{}
		} else {
			return errors.New(fmt.Sprintf("%s doubled,please check config file first", fp))
		}
	}
	return nil
}

func checkPathExisted(p string) error {
	_, err := os.Stat(p)
	return err
}
