const hre = require("hardhat");

const CONTRACT_ADDRESS = "0xb4B46bdAA835F8E4b4d8e208B6559cD267851051";

async function main() {
    const storage = await hre.ethers.getContractAt("Storage", CONTRACT_ADDRESS);

    console.log(`Conectado al contrato en: ${CONTRACT_ADDRESS}`);

    let currentValue = await storage.retrieve();
    console.log(`\nValor inicial almacenado: ${currentValue}`);


    console.log("\nEnviando transacción para guardar el número 777...");

    const tx = await storage.store(777);
    console.log(`Tx Hash: ${tx.hash}`);

    console.log("Esperando confirmación del bloque...");

    currentValue = await storage.retrieve();
    console.log(`\n¡Éxito! Nuevo valor almacenado: ${currentValue}`);
}

main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});