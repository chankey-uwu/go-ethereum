const { ethers } = require("ethers");

const RPC_URL = "http://127.0.0.1:32003"; 

const genesisAccounts = [
    { address: "0x8943545177806ED17B9F23F0a21ee5948eCaa776", pkey: "bcdf20249abf0ed6d944c0288fad489e33f66b3960d9e6229c1cd214ed3bbe31" },
    { address: "0xE25583099BA105D9ec0A67f5Ae86D90e50036425", pkey: "39725efee3fb28614de3bacaffe4cc4bd8c436257e2c8bb887c4b5c4be45e76d" },
];

const accountAddress = "0x23148FE43b3aEA23bb4fCb1b92A38d60451A3B23";

async function main() {
    const provider = new ethers.JsonRpcProvider(RPC_URL);
    
    try {
        const network = await provider.getNetwork();
        console.log(`Conectado a la red: ${network.chainId}`);
    } catch (e) {
        console.error("Error conectando. Revisa el puerto RPC de Kurtosis.");
        return;
    }

    const whaleWallet = new ethers.Wallet(genesisAccounts[0].pkey, provider);

    console.log(`\nCuenta Ballena usada para el envío: ${whaleWallet.address}`);

    const balanceBefore = await provider.getBalance(whaleWallet.address);
    console.log(`\nBalance de ${whaleWallet.address}:`);
    console.log(`${ethers.formatEther(balanceBefore)} ETH`);

    const amountToSend = ethers.parseEther("1.0"); 

    console.log(`\nEnviando 1.0 ETH a ${accountAddress}...`);
    
    const tx = await whaleWallet.sendTransaction({
        to: accountAddress,
        value: amountToSend
    });

    console.log(`Tx Hash: ${tx.hash}`);
    console.log("Esperando confirmación...");

    const receipt = await tx.wait();
    console.log("Transacción confirmada en el bloque.");

    const balanceAfter = await provider.getBalance(whaleWallet.address);
    console.log(`\nNuevo Balance de ${whaleWallet.address}:`);
    console.log(`${ethers.formatEther(balanceAfter)} ETH`);

    if (receipt){
        console.log("\n--- Receipt Completo (JSON) ---");
        console.log(JSON.stringify(receipt.toJSON(), null, 2));
        console.log("-------------------------------\n");
    }
}

main().catch((error) => {
    console.error(error);
    process.exit(1);
});