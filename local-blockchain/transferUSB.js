const { ethers } = require("ethers");

const RPC_URL = "http://127.0.0.1:32003"; 
const usbLocation = "E:/wallets/";

async function main() {
    const provider = new ethers.JsonRpcProvider(RPC_URL);
    
    try {
        const network = await provider.getNetwork();
        console.log(`Conectado a la red: ${network.chainId}`);
    } catch (e) {
        console.error("Error conectando. Revisa el puerto RPC de Kurtosis.");
        return;
    }

    const whaleWallet = new ethers.Wallet(require(usbLocation + "0x8943545177806ED17B9F23F0a21ee5948eCaa776.json").privateKey, provider);

    console.log(`\nCuenta Ballena usada para el envío: ${whaleWallet.address}`);

    const balanceBefore = await provider.getBalance(whaleWallet.address);
    console.log(`\nBalance de ${whaleWallet.address}:`);
    console.log(`${ethers.formatEther(balanceBefore)} ETH`);

    const amountToSend = ethers.parseEther("1000.0"); 

    const accountAddress = "0x7937b877102BAF25b735aB0D5f1addeC2893aaaD";

    console.log(`\nEnviando 1000.0 ETH a ${accountAddress}...`);
    
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