import neo4j from 'neo4j-driver'

/**
 * Creates driver for Neo4j database. 
 * NOTE: This does not connect to the database
 */
export const createDriver = () => {
    const driver = neo4j.driver(
        process.env.NEO4J_URI || 'bolt://localhost:7687',
        neo4j.auth.basic(
            process.env.NEO4J_USER || 'neo4j',
            process.env.NEO4J_PASSWORD || 'letmein'
        )
    )
    return driver
}