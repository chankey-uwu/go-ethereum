const { ethers } = require("ethers");

const RPC_URL = "http://127.0.0.1:32003";

const accountsToCheck = [
    "0x8943545177806ED17B9F23F0a21ee5948eCaa776",
    "0xE25583099BA105D9ec0A67f5Ae86D90e50036425",
    "0xccdd5b657ab09ba1cc7e126e06664e9241836733",
    "0x5e99D75F850f9118fCC711e64eD94E26f8F745f6",
    "0x23148FE43b3aEA23bb4fCb1b92A38d60451A3B23",
]

async function main() {
    const provider = new ethers.JsonRpcProvider(RPC_URL);
    
    try {
        const network = await provider.getNetwork();
        console.log(`Conectado a la red: ${network.chainId}`);
    } catch (e) {
        console.error("Error conectando. Revisa el puerto RPC de Kurtosis.");
        return;
    }

    for (const account of accountsToCheck) {
        const balance = await provider.getBalance(account);
        console.log(`Balance de ${account}: ${ethers.formatEther(balance)} ETH`);
    }
}

main().catch((error) => {
    console.error(error);
    process.exit(1);
});