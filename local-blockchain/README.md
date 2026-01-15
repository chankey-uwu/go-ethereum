# Blockchain local con Kurtosis

## Instrucciones

1. En `go-ethereum/` correr el siguiente comando para crear la imagen de geth en Docker (asegurarse de tener Docker Desktop abierto):
   ```console
   ../go-ethereum$ docker build -t <nombre_contenedor> .
   ```

2. Determinar los parámetros del archivo `network_params.yaml`, para más información: [Configuración de YAML](https://github.com/ethpandaops/ethereum-package?tab=readme-ov-file#configuration). Tiene que estar incluído `el_image: "<nombre_contenedor>"` para que use el geth forkeado.

3. Correr en la consola:
   ```console
   $ kurtosis run --enclave <nombre_enclave> github.com/ethpandaops/ethereum-package --args-file ./network_params.yaml --image-download always
   ```

