chia_transfer is a simple tool for transfering final plots from middle dir to final dir

## how install

```shell
# befor install chia_transfer,you need install go1.15.5 or later
git clone https://github.com/apehole/chia_transfer.git
cd chia_transfer
make all
sudo make install
```

## how run

```shell
# befor run you need an config file in homedir whic like deme_chia_transfer.yaml in chia_transfer dir
# you should write your middle path and final path into config file,and you need add path like /mnt/middle not /mnt/middle/,whitout right "/".(I was lazy 0.0)

nohup chia_transfer >> transfer.log &
```

## tips

This is a self-use tool that contains only the most basic transfer functions. There may be problems.

