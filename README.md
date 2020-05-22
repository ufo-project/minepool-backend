# Minepool-backend

How to deploy a private minerpool.

## Download files

Click on the link: <https://github.com/ufo-project/ufochain/releases>
Download the corresponding file according to the system and extract it.

## Create a wallet(If you hava keywords of wallet,skip this step)

Enter the unzipped folder,exec command:`ufo-wallet init`;

After entering and confirming the password, you will get a series of 12 English words separated by semicolons

Please make a good backup and keep it carefully to avoid being acquired by others.

    chalk;husband;exit;another;vessel;slam;federal;idle;horror;traffic;lobster;random;

After this step, a file called `wallet.db` is generated under the current path.

## Restore wallet database files using keywords(if already has wallet.db,skip this stop)

Execute `ufo-wallet restore --seed_phrase=keywords` (keywords separated by semicolon)

After this step, a file called `wallet.db` is generated under the current path.

## Start UFO node

(1) Need to export `secret subkey1` as miner key:

    Execute `ufo-wallet export_miner_key --subkey=1`,get `Secret Subkey 1` in result.

(2) Need to export `owner key`:

    Execute `ufo-wallet export_owner_key`,get `Owner Viewer key` in result.

(3) Start ufo node:

    Execute ufo-node --port=20001 --treasury_path=treasury.bin --stratum_port=20002 --stratum_use_tls=0 --mining_threads=0 --miner_key=MINERKEY --owner_key=OWNERKEY --pass=PASSWORD

Parameter Specification:
| Parameter | Implication |
| ---------:|:----:|
| port   |P2P listening port|
|stratum_port|stratum listening port,provides mining function|
|stratum_use_tls|Whether to use TLS to encrypt the transport channel, the stratum_secrets_path option also needs to be configured if enabled, and not if not|
|stratum_secrets_path|The corresponding catalog contains the TLS certificate file (stratum.crt), the key file corresponding to the TLS certificate (stratum.key), and the TLS API's key file (stratum.api.keys).This file is a text file with more than 8 bytes of characters, such as abcd1234|
|mining_threads|The number of mining threads is recommended to be set to the number of CPU cores|
|miner_key|Miner key exported from previous steps|
|owner_key|Owner key exported from previous steps|
|pass|Wallet password|

## Start Wallet API

(1) Enter the unzipped folder,exec command:`wallet-api.exe --node_addr=127.0.0.1:20001 --pass=12345678 --use_http=1`; 

| parameter | implication |
| ---------:|:----:|
| node_addr|ufo node ip address and p2p port|
|pass|ufo node set password|

## Config And Start UFOPool

(1) Downloadd minerpool source code from:<https://github.com/ufo-project/minepool-backend>

(2) Enter x17r path,execute:

    cmake .
    make
(3) Back to ufo-pool path,execute:

    go build

After this stop,will generated a file named ufo-pool under current path.


(4)Edit `config.json` file

`"WalletUrl": "127.0.0.1:20002"`    (ufo node ip address:node stratum port)

`"WalletApiUrl": "http://127.0.0.1:10000/api/wallet"`    (walet-api ip address:port)

(5)Start ufo pool

Execute:`./ufo-ppol`
