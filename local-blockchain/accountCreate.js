const { ethers } = require("ethers");

const RPC_URL = "http://127.0.0.1:32003";

async function main() {
    const provider = new ethers.JsonRpcProvider(RPC_URL);
    
    try {
        const network = await provider.getNetwork();
        console.log(`Conectado a la red: ${network.chainId}`);
    } catch (e) {
        console.error("Error conectando. Revisa el puerto RPC de Kurtosis.");
        return;
    }

    const wallet = ethers.Wallet.createRandom();
    console.log("Nueva cuenta creada:");
    console.log(`Dirección: ${wallet.address}`);
    console.log(`Clave Privada: ${wallet.privateKey}`);
}


main().catch((error) => {
    console.error(error);
    process.exit(1);
}); 