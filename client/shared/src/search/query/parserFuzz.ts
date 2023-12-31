export enum FilterType {
    repo = 'repo',
    repohasfile = 'repohasfile',
    repohascommitafter = 'repohascommitafter',
    file = 'file',
    type = 'type',
    case = 'case',
    lang = 'lang',
    fork = 'fork',
    archived = 'archived',
    visibility = 'visibility',
    count = 'count',
    timeout = 'timeout',
    before = 'before',
    after = 'after',
    author = 'author',
    committer = 'committer',
    message = 'message',
    content = 'content',
    patterntype = 'patterntype',
    index = 'index',
    // eslint-disable-next-line unicorn/prevent-abbreviations
    rev = 'rev',
}

/* eslint-disable unicorn/prevent-abbreviations */
export enum AliasedFilterType {
    r = 'repo',
    f = 'file',
    l = 'lang',
    language = 'lang',
    until = 'before',
    since = 'after',
    m = 'message',
    msg = 'message',
    revision = 'rev',
}
/* eslint-enable unicorn/prevent-abbreviations */

export const isFilterType = (filter: string): filter is FilterType => filter in FilterType
export const isAliasedFilterType = (filter: string): boolean => filter in AliasedFilterType

export const filterTypeKeys: FilterType[] = Object.keys(FilterType) as FilterType[]
export const filterTypeKeysWithAliases: (FilterType | AliasedFilterType)[] = [
    ...filterTypeKeys,
    ...Object.keys(AliasedFilterType),
] as (FilterType | AliasedFilterType)[]

export enum NegatedFilters {
    repo = '-repo',
    file = '-file',
    lang = '-lang',
    r = '-r',
    f = '-f',
    l = '-l',
    repohasfile = '-repohasfile',
    content = '-content',
    committer = '-committer',
    author = '-author',
    message = '-message',
}

/** The list of filters that are able to be negated. */
export type NegatableFilter =
    | FilterType.repo
    | FilterType.file
    | FilterType.repohasfile
    | FilterType.lang
    | FilterType.content
    | FilterType.committer
    | FilterType.author
    | FilterType.message

export const isNegatableFilter = (filter: FilterType): filter is NegatableFilter =>
    Object.keys(NegatedFilters).includes(filter)

/** The list of all negated filters. i.e. all valid filters that have `-` as a suffix. */
export const negatedFilters = Object.values(NegatedFilters)

export const isNegatedFilter = (filter: string): filter is NegatedFilters =>
    negatedFilters.includes(filter as NegatedFilters)

const negatedFilterToNegatableFilter: { [key: string]: NegatableFilter } = {
    '-repo': FilterType.repo,
    '-file': FilterType.file,
    '-lang': FilterType.lang,
    '-r': FilterType.repo,
    '-f': FilterType.file,
    '-l': FilterType.lang,
    '-repohasfile': FilterType.repohasfile,
    '-content': FilterType.content,
    '-committer': FilterType.committer,
    '-author': FilterType.author,
    '-message': FilterType.message,
}

export const resolveNegatedFilter = (filter: NegatedFilters): NegatableFilter => negatedFilterToNegatableFilter[filter]

/**
 * Represents a zero-indexed character range in a single-line search query.
 */
export interface CharacterRange {
    /** Zero-based character on the line */
    start: number
    /** Zero-based character on the line */
    end: number
}

export enum PatternKind {
    Literal = 1,
    Regexp,
    Structural,
}

export interface Pattern {
    type: 'pattern'
    range: CharacterRange
    kind: PatternKind
    value: string
}

/**
 * Represents a literal in a search query.
 *
 * Example: `Conn`.
 */
export interface Literal {
    type: 'literal'
    range: CharacterRange
    value: string
}

/**
 * Represents a filter in a search query.
 *
 * Example: `repo:^github\.com\/sourcegraph\/sourcegraph$`.
 */
export interface Filter {
    type: 'filter'
    range: CharacterRange
    filterType: Literal
    filterValue: Quoted | Literal | undefined
}

/**
 * Represents an operator in a search query.
 *
 * Example: AND, OR, NOT.
 */
export interface Operator {
    type: 'operator'
    range: CharacterRange
    value: string
}

/**
 * Represents a sequence of tokens in a search query.
 */
export interface Sequence {
    type: 'sequence'
    range: CharacterRange
    members: Token[]
}

/**
 * Represents a quoted string in a search query.
 *
 * Example: "Conn".
 */
export interface Quoted {
    type: 'quoted'
    range: CharacterRange
    quotedValue: string
}

/**
 * Represents a C-style comment, terminated by a newline.
 *
 * Example: `// Oh hai`
 */
export interface Comment {
    type: 'comment'
    range: CharacterRange
    value: string
}

export interface Whitespace {
    type: 'whitespace'
    range: CharacterRange
}

export interface OpeningParen {
    type: 'openingParen'
    range: CharacterRange
}

export interface ClosingParen {
    type: 'closingParen'
    range: CharacterRange
}

export type Token = Whitespace | OpeningParen | ClosingParen | Operator | Comment | Literal | Pattern | Filter | Quoted

export type Term = Token | Sequence

/**
 * Represents the failed result of running a {@link Parser} on a search query.
 */
interface ParseError {
    type: 'error'

    /**
     * A string representing the token that would have been expected
     * for successful parsing at {@link ParseError#at}.
     */
    expected: string

    /**
     * The index in the search query string where parsing failed.
     */
    at: number
}

/**
 * Represents the successful result of running a {@link Parser} on a search query.
 */
export interface ParseSuccess<T = Term> {
    type: 'success'

    /**
     * The resulting term.
     */
    token: T
}

/**
 * Represents the result of running a {@link Parser} on a search query.
 */
export type ParserResult<T = Term> = ParseError | ParseSuccess<T>

type Parser<T = Term> = (input: string, start: number) => ParserResult<T>

/**
 * Returns a {@link Parser} that succeeds if zero or more tokens parsed
 * by the given `parseToken` parsers are found in a search query.
 */
const zeroOrMore =
    (parseToken: Parser<Term>): Parser<Sequence> =>
    (input, start) => {
        const members: Token[] = []
        let adjustedStart = start
        let end = start + 1
        while (input[adjustedStart] !== undefined) {
            const result = parseToken(input, adjustedStart)
            if (result.type === 'error') {
                return result
            }
            if (result.token.type === 'sequence') {
                for (const member of result.token.members) {
                    members.push(member)
                }
            } else {
                members.push(result.token)
            }
            end = result.token.range.end
            adjustedStart = end
        }
        return {
            type: 'success',
            token: { type: 'sequence', members, range: { start, end } },
        }
    }

/**
 * Returns a {@link Parser} that succeeds if any of the given parsers succeeds.
 */
const oneOf =
    <T>(...parsers: Parser<T>[]): Parser<T> =>
    (input, start) => {
        const expected: string[] = []
        for (const parser of parsers) {
            const result = parser(input, start)
            if (result.type === 'success') {
                return result
            }
            expected.push(result.expected)
        }
        return {
            type: 'error',
            expected: `One of: ${expected.join(', ')}`,
            at: start,
        }
    }

/**
 * A {@link Parser} that will attempt to parse delimited strings for an arbitrary
 * delimiter. `\` is treated as an escape character for the delimited string.
 */
const quoted =
    (delimiter: string): Parser<Quoted> =>
    (input, start) => {
        if (input[start] !== delimiter) {
            return { type: 'error', expected: delimiter, at: start }
        }
        let end = start + 1
        while (input[end] && (input[end] !== delimiter || input[end - 1] === '\\')) {
            end = end + 1
        }
        if (!input[end]) {
            return { type: 'error', expected: delimiter, at: end }
        }
        return {
            type: 'success',
            // end + 1 as `end` is currently the index of the quote in the string.
            token: { type: 'quoted', quotedValue: input.slice(start + 1, end), range: { start, end: end + 1 } },
        }
    }

/**
 * Returns a {@link Parser} that will attempt to parse tokens matching
 * the given character in a search query.
 */
const character =
    (character: string): Parser<Literal> =>
    (input, start) => {
        if (input[start] !== character) {
            return { type: 'error', expected: character, at: start }
        }
        return {
            type: 'success',
            token: { type: 'literal', value: character, range: { start, end: start + 1 } },
        }
    }

/**
 * Returns a {@link Parser} that will attempt to parse
 * tokens matching the given RegExp pattern in a search query.
 */
const scanToken = <T extends Term = Literal>(
    regexp: RegExp,
    output?: T | ((input: string, range: CharacterRange) => T),
    expected?: string
): Parser<T> => {
    if (!regexp.source.startsWith('^')) {
        regexp = new RegExp(`^${regexp.source}`, regexp.flags)
    }
    return (input, start) => {
        const matchTarget = input.slice(Math.max(0, start))
        if (!matchTarget) {
            return { type: 'error', expected: expected || `/${regexp.source}/`, at: start }
        }
        const match = matchTarget.match(regexp)
        if (!match) {
            return { type: 'error', expected: expected || `/${regexp.source}/`, at: start }
        }
        const range = { start, end: start + match[0].length }
        return {
            type: 'success',
            token: output
                ? typeof output === 'function'
                    ? output(input, range)
                    : output
                : ({ type: 'literal', value: match[0], range } as T),
        }
    }
}

const whitespace = scanToken(
    /\s+/,
    (_input, range): Whitespace => ({
        type: 'whitespace',
        range,
    })
)

const literal = scanToken(/[^\s)]+/)

const operator = scanToken(
    /(and|AND|or|OR|not|NOT)/,
    (input, { start, end }): Operator => ({ type: 'operator', value: input.slice(start, end), range: { start, end } })
)

const comment = scanToken(
    /\/\/.*/,
    (input, { start, end }): Comment => ({ type: 'comment', value: input.slice(start, end), range: { start, end } })
)

const filterKeyword = scanToken(new RegExp(`-?(${filterTypeKeysWithAliases.join('|')})+(?=:)`, 'i'))

const filterDelimiter = character(':')

const filterValue = oneOf<Quoted | Literal>(quoted('"'), quoted("'"), literal)

const openingParen = scanToken(/\(/, (_input, range): OpeningParen => ({ type: 'openingParen', range }))

const closingParen = scanToken(/\)/, (_input, range): ClosingParen => ({ type: 'closingParen', range }))

/**
 * Returns a {@link Parser} that succeeds if a token parsed by `parseToken`,
 * followed by whitespace or EOF, is found in the search query.
 */
const followedBy =
    (parseToken: Parser<Token>, parseNext: Parser<Token>): Parser<Sequence> =>
    (input, start) => {
        const members: Token[] = []
        const tokenResult = parseToken(input, start)
        if (tokenResult.type === 'error') {
            return tokenResult
        }
        members.push(tokenResult.token)
        let { end } = tokenResult.token.range
        if (input[end] !== undefined) {
            const separatorResult = parseNext(input, end)
            if (separatorResult.type === 'error') {
                return separatorResult
            }
            members.push(separatorResult.token)
            end = separatorResult.token.range.end
        }
        return {
            type: 'success',
            token: { type: 'sequence', members, range: { start, end } },
        }
    }

/**
 * A {@link Parser} that will attempt to parse {@link Filter} tokens
 * (consisting a of a filter type and a filter value, separated by a colon)
 * in a search query.
 */
const filter: Parser<Filter> = (input, start) => {
    const parsedKeyword = filterKeyword(input, start)
    if (parsedKeyword.type === 'error') {
        return parsedKeyword
    }
    const parsedDelimiter = filterDelimiter(input, parsedKeyword.token.range.end)
    if (parsedDelimiter.type === 'error') {
        return parsedDelimiter
    }
    const parsedValue =
        input[parsedDelimiter.token.range.end] === undefined
            ? undefined
            : filterValue(input, parsedDelimiter.token.range.end)
    if (parsedValue && parsedValue.type === 'error') {
        return parsedValue
    }
    return {
        type: 'success',
        token: {
            type: 'filter',
            range: { start, end: parsedValue ? parsedValue.token.range.end : parsedDelimiter.token.range.end },
            filterType: parsedKeyword.token,
            filterValue: parsedValue?.token,
        },
    }
}

const createPattern = (value: string, range: CharacterRange, kind: PatternKind): ParseSuccess<Pattern> => ({
    type: 'success',
    token: {
        type: 'pattern',
        range,
        kind,
        value,
    },
})

const scanFilterOrOperator = oneOf<Literal | Sequence>(filterKeyword, followedBy(operator, whitespace))
const keepScanning = (input: string, start: number): boolean => scanFilterOrOperator(input, start).type !== 'success'

/**
 * ScanBalancedPattern attempts to scan balanced parentheses as literal patterns. This
 * ensures that we interpret patterns containing parentheses _as patterns_ and not
 * groups. For example, it accepts these patterns:
 *
 * ((a|b)|c)              - a regular expression with balanced parentheses for grouping
 * myFunction(arg1, arg2) - a literal string with parens that should be literally interpreted
 * foo(...)               - a structural search pattern
 *
 * If it weren't for this scanner, the above parentheses would have to be
 * interpreted as part of the query language group syntax, like these:
 *
 * (foo or (bar and baz))
 *
 * So, this scanner detects parentheses as patterns without needing the user to
 * explicitly escape them. As such, there are cases where this scanner should
 * not succeed:
 *
 * (foo or (bar and baz)) - a valid query with and/or expression groups in the query langugae
 * (repo:foo bar baz)     - a valid query containing a recognized repo: field. Here parentheses are interpreted as a group, not a pattern.
 */
export const scanBalancedPattern =
    (kind = PatternKind.Literal): Parser<Pattern> =>
    (input, start) => {
        let adjustedStart = start
        let balanced = 0
        let current = ''
        const result: string[] = []

        const nextChar = (): void => {
            current = input[adjustedStart]
            adjustedStart += 1
        }

        if (!keepScanning(input, start)) {
            return {
                type: 'error',
                expected: 'no recognized filter or operator',
                at: start,
            }
        }

        while (input[adjustedStart] !== undefined) {
            nextChar()
            if (current === ' ' && balanced === 0) {
                // Stop scanning a potential pattern when we see whitespace in a balanced state.
                adjustedStart -= 1 // Backtrack.
                break
            } else if (current === '(') {
                if (!keepScanning(input, adjustedStart)) {
                    return {
                        type: 'error',
                        expected: 'no recognized filter or operator',
                        at: adjustedStart,
                    }
                }
                balanced += 1
                result.push(current)
            } else if (current === ')') {
                balanced -= 1
                if (balanced < 0) {
                    // This paren is an unmatched closing paren, so we stop treating it as a potential
                    // pattern here--it might be closing a group.
                    adjustedStart -= 1 // Backtrack.
                    balanced = 0 // Pattern is balanced up to this point
                    break
                }
                result.push(current)
            } else if (current === ' ') {
                if (!keepScanning(input, adjustedStart)) {
                    return {
                        type: 'error',
                        expected: 'no recognized filter or operator',
                        at: adjustedStart,
                    }
                }
                result.push(current)
            } else if (current === '\\') {
                if (input[adjustedStart] !== undefined) {
                    nextChar()
                    // Accept anything anything escaped. The point is to consume escaped spaces like "\ "
                    // so that we don't recognize it as terminating a pattern.
                    result.push('\\', current)
                    continue
                }
                result.push(current)
            } else {
                result.push(current)
            }
        }

        if (balanced !== 0) {
            return {
                type: 'error',
                expected: 'no unbalanced parentheses',
                at: adjustedStart,
            }
        }

        return createPattern(result.join(''), { start, end: adjustedStart }, kind)
    }

const scanPattern =
    (kind: PatternKind): Parser<Pattern> =>
    (input, start) => {
        const balancedPattern = scanBalancedPattern(kind)(input, start)
        if (balancedPattern.type === 'success') {
            return createPattern(balancedPattern.token.value, balancedPattern.token.range, kind)
        }

        const anyPattern = literal(input, start)
        if (anyPattern.type === 'success') {
            return createPattern(anyPattern.token.value, anyPattern.token.range, kind)
        }

        return anyPattern
    }

const whitespaceOrClosingParen = oneOf<Whitespace | ClosingParen>(whitespace, closingParen)

/**
 * A {@link Parser} for a Sourcegraph search query, interpreting patterns for {@link PatternKind}.
 *
 * @param interpretComments Interpets C-style line comments for multiline queries.
 */
const createParser = (kind: PatternKind, interpretComments?: boolean): Parser<Sequence> => {
    const baseQuotedScanner = [quoted('"'), quoted("'")]
    const quotedScanner = kind === PatternKind.Regexp ? [quoted('/'), ...baseQuotedScanner] : baseQuotedScanner

    const baseScanner = [operator, filter, ...quotedScanner, scanPattern(kind)]
    const tokenScanner: Parser<Token>[] = interpretComments ? [comment, ...baseScanner] : baseScanner

    const baseEarlyPatternScanner = [...quotedScanner, scanBalancedPattern(kind)]
    const earlyPatternScanner = interpretComments ? [comment, ...baseEarlyPatternScanner] : baseEarlyPatternScanner

    return zeroOrMore(
        oneOf<Term>(
            whitespace,
            ...earlyPatternScanner.map(token => followedBy(token, whitespaceOrClosingParen)),
            openingParen,
            closingParen,
            ...tokenScanner.map(token => followedBy(token, whitespaceOrClosingParen))
        )
    )
}

/**
 * Parses a search query string.
 */
export const parseSearchQuery = (
    query: string,
    interpretComments?: boolean,
    kind = PatternKind.Literal
): ParserResult<Sequence> => {
    const scanner = createParser(kind, interpretComments)
    return scanner(query, 0)
}
