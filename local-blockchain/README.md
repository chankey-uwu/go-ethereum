# Blockchain local con Kurtosis

## Instrucciones

1. En `go-ethereum/` correr el siguiente comando para crear la imagen de geth en Docker (asegurarse de tener Docker Desktop abierto):
   ```console
   ../go-ethereum$ docker build -t geth-usm:v1 .
   ```

2. Determinar los parámetros del archivo `network_params.yaml`, para más información: [Configuración de YAML](https://github.com/ethpandaops/ethereum-package?tab=readme-ov-file#configuration). Tiene que estar incluído `el_image: "geth-usm:v1"` para que use el geth forkeado.

3. Correr en la consola:
   ```console
   ../go-ethereum/local-blockchain$ kurtosis run --enclave local-blockchain github.com/ethpandaops/ethereum-package --args-file ./network_params.yaml
   ```

## Comandos Kurtosis

1. `kurtosis enclave inspect local-blockchain`: devuelve la lista de todos los servicios junto sus respectivos puertos.
2. `kurtosis enclave stop local-blockchain`: detiene la ejecución de la Blockchain. ADVERTENCIA: No se puede volver a reanudar.
3. `kurtosis enclave rm -f local-blockchain`: elimina los archivos de la Blockchain.
4. `kurtosis engine restart`: reinicia el motor de kurtosis. Sirve porque a veces quedan Encloves eliminados aún siendo listados o se desconecta de Docker.
5. `kurtosis service shell local-network el-x-geth-lighthouse`: abre una terminal para el servicio **x**.

## Archivos adicionales

1. `transfer.js`: realiza una transacción de 1 ETH desde una cuenta "ballena" a una cuenta a elección.
2. `checkBalance.js`: revisa el saldo de las cuentas (no todas, las que están en una lista).
3. `accountCreate.js`: crea una cuenta e imprime su dirección y llave privada.

## Dora

Abriendo `http://127.0.0.1:35000` en el navegador se puede ver un Dashboard de la Blockchain, como las épocas, bloques

## Nodo local

1. En `go-ethereum` correr el siguiente comando para inicializar **geth** localmente:
   ```console
   ../go-ethereum$ ./build/bin/geth init --datadir ./local-blockchain/local-node ./local-blockchain/genesis-data/geth-genesis.json
   ```

2. Luego, para inicializar el nodo y que se conecte a la red corremos en `go-ethereum`:
    ```console
    ../go-ethereum$ ./build/bin/geth --datadir .\local-blockchain\local-node --networkid 585858 --http --http.port 8547 --http.api "eth,net,web3,admin" --port 30304 --bootnodes "enode://16a02395758b437b65205c8f79ba35b7763a26c2d6091ba9140b8bcc5b5225a00cace4a818df1678f302c9d8cb568998567eee01bf20ffcdb792d9a91ff96c58@127.0.0.1:32000" --authrpc.port 8553
    ```

3. Para abrir una terminal en **geth**:
   ```console
   ../go-ethereum$ ./build/bin/geth attach http://127.0.0.1:8547
   ```

   y luego correr:

   ```console
   > admin.addPeer("enode://16a02395758b437b65205c8f79ba35b7763a26c2d6091ba9140b8bcc5b5225a00cace4a818df1678f302c9d8cb568998567eee01bf20ffcdb792d9a91ff96c58@127.0.0.1:32000")
   ```

