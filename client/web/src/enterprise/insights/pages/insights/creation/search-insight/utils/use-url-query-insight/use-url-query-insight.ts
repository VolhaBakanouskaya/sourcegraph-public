import { useMemo } from 'react'

import { dedupeWhitespace } from '@sourcegraph/common'
import { FilterType } from '@sourcegraph/shared/src/search/query/filters'
import { stringHuman } from '@sourcegraph/shared/src/search/query/printer'
import { scanSearchQuery } from '@sourcegraph/shared/src/search/query/scanner'
import { isFilterType, isRepoFilter } from '@sourcegraph/shared/src/search/query/validate'

import { createDefaultEditSeries } from '../../../../../../components'
import { CreateInsightFormFields } from '../../types'

export type UseURLQueryInsightResult = Partial<CreateInsightFormFields> | null

/**
 * Returns initial values for the search insight from query param.
 */
export function useURLQueryInsight(queryParameters: string): UseURLQueryInsightResult {
    const query = useMemo(() => new URLSearchParams(queryParameters).get('query'), [queryParameters])

    return useMemo(() => {
        if (!query) {
            return null
        }

        const { seriesQuery, repoQuery } = getInsightDataFromQuery(query)

        return {
            series: [
                createDefaultEditSeries({
                    edit: true,
                    valid: true,
                    name: 'Search series #1',
                    query: seriesQuery ?? '',
                }),
            ],
            repoMode: repoQuery ? 'search-query' : 'urls-list',
            repoQuery: { query: repoQuery },
        }
    }, [query])
}

export interface InsightData {
    repoQuery: string
    seriesQuery: string
}

/**
 * Return serialized value of repositories and query from the URL query params.
 *
 * @param searchQuery -- query param with possible value for the creation UI
 *
 * Exported for testing only.
 */
export function getInsightDataFromQuery(searchQuery: string | null): InsightData {
    const sequence = scanSearchQuery(searchQuery ?? '')

    if (!searchQuery || sequence.type === 'error') {
        return {
            seriesQuery: '',
            repoQuery: '',
        }
    }

    const tokens = Array.isArray(sequence.term) ? sequence.term : [sequence.term]

    // Generate a string query from tokens with repo: and context filters only
    // in order to put this in the repo query field
    const tokensWithRepoFiltersAndContext = tokens.filter(
        token => isRepoFilter(token) || isFilterType(token, FilterType.context) || token.type === 'whitespace'
    )

    // Generate a string query from tokens without repo: filters for the insight
    // query field.
    const tokensWithoutRepoFiltersAndContext = tokens.filter(
        token => !isRepoFilter(token) && !isFilterType(token, FilterType.context)
    )

    return {
        seriesQuery: dedupeWhitespace(stringHuman(tokensWithoutRepoFiltersAndContext)),
        repoQuery: dedupeWhitespace(stringHuman(tokensWithRepoFiltersAndContext)),
    }
}
