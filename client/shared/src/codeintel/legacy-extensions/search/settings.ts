/* tslint:disable */
/**
 * This file was automatically generated by json-schema-to-typescript.
 * DO NOT MODIFY IT BY HAND. Instead, modify the source JSONSchema file,
 * and run json-schema-to-typescript to regenerate this file.
 */

export interface SearchBasedCodeIntelligenceSettings {
    /**
     * Whether to enable trace logging on the extension.
     */
    'codeIntel.traceExtension'?: boolean
    /**
     * Whether to fetch multiple precise definitions and references on hover.
     */
    'codeIntel.disableRangeQueries'?: boolean
    /**
     * Whether to supplement precise references with search-based results.
     */
    'codeIntel.mixPreciseAndSearchBasedReferences'?: boolean
    /**
     * Whether to include forked repositories in search results.
     */
    'basicCodeIntel.includeForks'?: boolean
    /**
     * Whether to include archived repositories in search results.
     */
    'basicCodeIntel.includeArchives'?: boolean
    /**
     * Whether to run global searches over all repositories.
     */
    'basicCodeIntel.globalSearchesEnabled'?: boolean
    /**
     * Whether to use only indexed requests to the search API.
     */
    'basicCodeIntel.indexOnly'?: boolean
    /**
     * The timeout (in milliseconds) for un-indexed search requests.
     */
    'basicCodeIntel.unindexedSearchTimeout'?: number
    /**
     * Whether to request the `fileLocal` field in GraphQL. It's unavailable in older versions of Sourcegraph. Defaults to false.
     */
    fileLocal?: boolean
}
