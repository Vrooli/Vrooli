export const projectOrOrganization_findMany = {
  "fieldName": "projectOrOrganizations",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectOrOrganizations",
        "loc": {
          "start": 2104,
          "end": 2126
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2127,
              "end": 2132
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2135,
                "end": 2140
              }
            },
            "loc": {
              "start": 2134,
              "end": 2140
            }
          },
          "loc": {
            "start": 2127,
            "end": 2140
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "edges",
              "loc": {
                "start": 2148,
                "end": 2153
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "cursor",
                    "loc": {
                      "start": 2164,
                      "end": 2170
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2164,
                    "end": 2170
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 2179,
                      "end": 2183
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Project",
                            "loc": {
                              "start": 2205,
                              "end": 2212
                            }
                          },
                          "loc": {
                            "start": 2205,
                            "end": 2212
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Project_list",
                                "loc": {
                                  "start": 2234,
                                  "end": 2246
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2231,
                                "end": 2246
                              }
                            }
                          ],
                          "loc": {
                            "start": 2213,
                            "end": 2260
                          }
                        },
                        "loc": {
                          "start": 2198,
                          "end": 2260
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Organization",
                            "loc": {
                              "start": 2280,
                              "end": 2292
                            }
                          },
                          "loc": {
                            "start": 2280,
                            "end": 2292
                          }
                        },
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Organization_list",
                                "loc": {
                                  "start": 2314,
                                  "end": 2331
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2311,
                                "end": 2331
                              }
                            }
                          ],
                          "loc": {
                            "start": 2293,
                            "end": 2345
                          }
                        },
                        "loc": {
                          "start": 2273,
                          "end": 2345
                        }
                      }
                    ],
                    "loc": {
                      "start": 2184,
                      "end": 2355
                    }
                  },
                  "loc": {
                    "start": 2179,
                    "end": 2355
                  }
                }
              ],
              "loc": {
                "start": 2154,
                "end": 2361
              }
            },
            "loc": {
              "start": 2148,
              "end": 2361
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 2366,
                "end": 2374
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 2385,
                      "end": 2396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2385,
                    "end": 2396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 2405,
                      "end": 2421
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2405,
                    "end": 2421
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorOrganization",
                    "loc": {
                      "start": 2430,
                      "end": 2451
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2430,
                    "end": 2451
                  }
                }
              ],
              "loc": {
                "start": 2375,
                "end": 2457
              }
            },
            "loc": {
              "start": 2366,
              "end": 2457
            }
          }
        ],
        "loc": {
          "start": 2142,
          "end": 2461
        }
      },
      "loc": {
        "start": 2104,
        "end": 2461
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 32,
          "end": 34
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 34
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 35,
          "end": 45
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 35,
        "end": 45
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 46,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 46,
        "end": 56
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 57,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 57,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 63,
          "end": 68
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 68
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 69,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 88,
                  "end": 100
                }
              },
              "loc": {
                "start": 88,
                "end": 100
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 114,
                      "end": 130
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 111,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 101,
                "end": 136
              }
            },
            "loc": {
              "start": 81,
              "end": 136
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 148,
                  "end": 152
                }
              },
              "loc": {
                "start": 148,
                "end": 152
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 166,
                      "end": 174
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 174
                  }
                }
              ],
              "loc": {
                "start": 153,
                "end": 180
              }
            },
            "loc": {
              "start": 141,
              "end": 180
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 182
        }
      },
      "loc": {
        "start": 69,
        "end": 182
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 183,
          "end": 186
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 193,
                "end": 202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 207,
                "end": 216
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 207,
              "end": 216
            }
          }
        ],
        "loc": {
          "start": 187,
          "end": 218
        }
      },
      "loc": {
        "start": 183,
        "end": 218
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 266,
          "end": 268
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 266,
        "end": 268
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 269,
          "end": 280
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 269,
        "end": 280
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 281,
          "end": 287
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 281,
        "end": 287
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 288,
          "end": 298
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 288,
        "end": 298
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 299,
          "end": 309
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 299,
        "end": 309
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 310,
          "end": 328
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 310,
        "end": 328
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 329,
          "end": 338
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 329,
        "end": 338
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 339,
          "end": 352
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 339,
        "end": 352
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 353,
          "end": 365
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 353,
        "end": 365
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 366,
          "end": 378
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 366,
        "end": 378
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 379,
          "end": 391
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 379,
        "end": 391
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 392,
          "end": 401
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 392,
        "end": 401
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 402,
          "end": 406
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 416,
                "end": 424
              }
            },
            "directives": [],
            "loc": {
              "start": 413,
              "end": 424
            }
          }
        ],
        "loc": {
          "start": 407,
          "end": 426
        }
      },
      "loc": {
        "start": 402,
        "end": 426
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 427,
          "end": 439
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 446,
                "end": 448
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 446,
              "end": 448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 453,
                "end": 461
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 453,
              "end": 461
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 466,
                "end": 469
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 466,
              "end": 469
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 474,
                "end": 478
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 474,
              "end": 478
            }
          }
        ],
        "loc": {
          "start": 440,
          "end": 480
        }
      },
      "loc": {
        "start": 427,
        "end": 480
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 481,
          "end": 484
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 491,
                "end": 504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 491,
              "end": 504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 509,
                "end": 518
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 509,
              "end": 518
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 523,
                "end": 534
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 523,
              "end": 534
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 539,
                "end": 548
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 539,
              "end": 548
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 553,
                "end": 562
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 553,
              "end": 562
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 567,
                "end": 574
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 567,
              "end": 574
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 579,
                "end": 591
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 579,
              "end": 591
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 596,
                "end": 604
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 596,
              "end": 604
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 609,
                "end": 623
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 634,
                      "end": 636
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 634,
                    "end": 636
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 645,
                      "end": 655
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 645,
                    "end": 655
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 664,
                      "end": 674
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 664,
                    "end": 674
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 683,
                      "end": 690
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 683,
                    "end": 690
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 699,
                      "end": 710
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 699,
                    "end": 710
                  }
                }
              ],
              "loc": {
                "start": 624,
                "end": 716
              }
            },
            "loc": {
              "start": 609,
              "end": 716
            }
          }
        ],
        "loc": {
          "start": 485,
          "end": 718
        }
      },
      "loc": {
        "start": 481,
        "end": 718
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 765,
          "end": 767
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 765,
        "end": 767
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 768,
          "end": 779
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 768,
        "end": 779
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 780,
          "end": 786
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 780,
        "end": 786
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 787,
          "end": 799
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 787,
        "end": 799
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 800,
          "end": 803
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 810,
                "end": 823
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 810,
              "end": 823
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 828,
                "end": 837
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 828,
              "end": 837
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 842,
                "end": 853
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 842,
              "end": 853
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 858,
                "end": 867
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 858,
              "end": 867
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 872,
                "end": 881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 872,
              "end": 881
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 886,
                "end": 893
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 886,
              "end": 893
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 898,
                "end": 910
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 898,
              "end": 910
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 915,
                "end": 923
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 915,
              "end": 923
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 928,
                "end": 942
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 953,
                      "end": 955
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 953,
                    "end": 955
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 964,
                      "end": 974
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 964,
                    "end": 974
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 983,
                      "end": 993
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 983,
                    "end": 993
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 1002,
                      "end": 1009
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1002,
                    "end": 1009
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 1018,
                      "end": 1029
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1018,
                    "end": 1029
                  }
                }
              ],
              "loc": {
                "start": 943,
                "end": 1035
              }
            },
            "loc": {
              "start": 928,
              "end": 1035
            }
          }
        ],
        "loc": {
          "start": 804,
          "end": 1037
        }
      },
      "loc": {
        "start": 800,
        "end": 1037
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1075,
          "end": 1083
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1090,
                "end": 1102
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1113,
                      "end": 1115
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1113,
                    "end": 1115
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1124,
                      "end": 1132
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1124,
                    "end": 1132
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1141,
                      "end": 1152
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1141,
                    "end": 1152
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1161,
                      "end": 1165
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1161,
                    "end": 1165
                  }
                }
              ],
              "loc": {
                "start": 1103,
                "end": 1171
              }
            },
            "loc": {
              "start": 1090,
              "end": 1171
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1176,
                "end": 1178
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1176,
              "end": 1178
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1183,
                "end": 1193
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1183,
              "end": 1193
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1198,
                "end": 1208
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1198,
              "end": 1208
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 1213,
                "end": 1229
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1213,
              "end": 1229
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1234,
                "end": 1242
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1234,
              "end": 1242
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1247,
                "end": 1256
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1247,
              "end": 1256
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1261,
                "end": 1273
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1261,
              "end": 1273
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 1278,
                "end": 1294
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1278,
              "end": 1294
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 1299,
                "end": 1309
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1299,
              "end": 1309
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1314,
                "end": 1326
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1314,
              "end": 1326
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1331,
                "end": 1343
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1331,
              "end": 1343
            }
          }
        ],
        "loc": {
          "start": 1084,
          "end": 1345
        }
      },
      "loc": {
        "start": 1075,
        "end": 1345
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1346,
          "end": 1348
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1346,
        "end": 1348
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1349,
          "end": 1359
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1349,
        "end": 1359
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1360,
          "end": 1370
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1360,
        "end": 1370
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1371,
          "end": 1380
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1371,
        "end": 1380
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1381,
          "end": 1392
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1381,
        "end": 1392
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1393,
          "end": 1399
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 1409,
                "end": 1419
              }
            },
            "directives": [],
            "loc": {
              "start": 1406,
              "end": 1419
            }
          }
        ],
        "loc": {
          "start": 1400,
          "end": 1421
        }
      },
      "loc": {
        "start": 1393,
        "end": 1421
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1422,
          "end": 1427
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1441,
                  "end": 1453
                }
              },
              "loc": {
                "start": 1441,
                "end": 1453
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1467,
                      "end": 1483
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1464,
                    "end": 1483
                  }
                }
              ],
              "loc": {
                "start": 1454,
                "end": 1489
              }
            },
            "loc": {
              "start": 1434,
              "end": 1489
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1501,
                  "end": 1505
                }
              },
              "loc": {
                "start": 1501,
                "end": 1505
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1519,
                      "end": 1527
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1516,
                    "end": 1527
                  }
                }
              ],
              "loc": {
                "start": 1506,
                "end": 1533
              }
            },
            "loc": {
              "start": 1494,
              "end": 1533
            }
          }
        ],
        "loc": {
          "start": 1428,
          "end": 1535
        }
      },
      "loc": {
        "start": 1422,
        "end": 1535
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1536,
          "end": 1547
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1536,
        "end": 1547
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1548,
          "end": 1562
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1548,
        "end": 1562
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1563,
          "end": 1568
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1563,
        "end": 1568
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1569,
          "end": 1578
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1569,
        "end": 1578
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1579,
          "end": 1583
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 1593,
                "end": 1601
              }
            },
            "directives": [],
            "loc": {
              "start": 1590,
              "end": 1601
            }
          }
        ],
        "loc": {
          "start": 1584,
          "end": 1603
        }
      },
      "loc": {
        "start": 1579,
        "end": 1603
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1604,
          "end": 1618
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1604,
        "end": 1618
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1619,
          "end": 1624
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1619,
        "end": 1624
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1625,
          "end": 1628
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1635,
                "end": 1644
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1635,
              "end": 1644
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1649,
                "end": 1660
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1649,
              "end": 1660
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1665,
                "end": 1676
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1665,
              "end": 1676
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1681,
                "end": 1690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1681,
              "end": 1690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1695,
                "end": 1702
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1695,
              "end": 1702
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1707,
                "end": 1715
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1707,
              "end": 1715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1720,
                "end": 1732
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1720,
              "end": 1732
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1737,
                "end": 1745
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1737,
              "end": 1745
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1750,
                "end": 1758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1750,
              "end": 1758
            }
          }
        ],
        "loc": {
          "start": 1629,
          "end": 1760
        }
      },
      "loc": {
        "start": 1625,
        "end": 1760
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1790,
          "end": 1792
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1790,
        "end": 1792
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1793,
          "end": 1803
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1793,
        "end": 1803
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 1804,
          "end": 1807
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1804,
        "end": 1807
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1808,
          "end": 1817
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1808,
        "end": 1817
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1818,
          "end": 1830
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1837,
                "end": 1839
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1837,
              "end": 1839
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1844,
                "end": 1852
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1844,
              "end": 1852
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1857,
                "end": 1868
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1857,
              "end": 1868
            }
          }
        ],
        "loc": {
          "start": 1831,
          "end": 1870
        }
      },
      "loc": {
        "start": 1818,
        "end": 1870
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1871,
          "end": 1874
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 1881,
                "end": 1886
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1881,
              "end": 1886
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1891,
                "end": 1903
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1891,
              "end": 1903
            }
          }
        ],
        "loc": {
          "start": 1875,
          "end": 1905
        }
      },
      "loc": {
        "start": 1871,
        "end": 1905
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1936,
          "end": 1938
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1936,
        "end": 1938
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1939,
          "end": 1949
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1939,
        "end": 1949
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1950,
          "end": 1960
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1950,
        "end": 1960
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1961,
          "end": 1972
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1961,
        "end": 1972
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1973,
          "end": 1979
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1973,
        "end": 1979
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 1980,
          "end": 1985
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1980,
        "end": 1985
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBotDepictingPerson",
        "loc": {
          "start": 1986,
          "end": 2006
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1986,
        "end": 2006
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 2007,
          "end": 2011
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2007,
        "end": 2011
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2012,
          "end": 2024
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2012,
        "end": 2024
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 10,
          "end": 20
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 24,
            "end": 29
          }
        },
        "loc": {
          "start": 24,
          "end": 29
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 32,
                "end": 34
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 34
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 35,
                "end": 45
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 35,
              "end": 45
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 46,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 46,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 57,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 69,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 88,
                        "end": 100
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 100
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 114,
                            "end": 130
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 111,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 101,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 136
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 148,
                        "end": 152
                      }
                    },
                    "loc": {
                      "start": 148,
                      "end": 152
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 166,
                            "end": 174
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 163,
                          "end": 174
                        }
                      }
                    ],
                    "loc": {
                      "start": 153,
                      "end": 180
                    }
                  },
                  "loc": {
                    "start": 141,
                    "end": 180
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 182
              }
            },
            "loc": {
              "start": 69,
              "end": 182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 183,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 193,
                      "end": 202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 207,
                      "end": 216
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 207,
                    "end": 216
                  }
                }
              ],
              "loc": {
                "start": 187,
                "end": 218
              }
            },
            "loc": {
              "start": 183,
              "end": 218
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 220
        }
      },
      "loc": {
        "start": 1,
        "end": 220
      }
    },
    "Organization_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_list",
        "loc": {
          "start": 230,
          "end": 247
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 251,
            "end": 263
          }
        },
        "loc": {
          "start": 251,
          "end": 263
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 266,
                "end": 268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 266,
              "end": 268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 269,
                "end": 280
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 269,
              "end": 280
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 281,
                "end": 287
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 281,
              "end": 287
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 288,
                "end": 298
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 288,
              "end": 298
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 299,
                "end": 309
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 299,
              "end": 309
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 310,
                "end": 328
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 310,
              "end": 328
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 329,
                "end": 338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 329,
              "end": 338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 339,
                "end": 352
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 339,
              "end": 352
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 353,
                "end": 365
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 353,
              "end": 365
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 366,
                "end": 378
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 366,
              "end": 378
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 379,
                "end": 391
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 379,
              "end": 391
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 392,
                "end": 401
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 392,
              "end": 401
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 402,
                "end": 406
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 416,
                      "end": 424
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 413,
                    "end": 424
                  }
                }
              ],
              "loc": {
                "start": 407,
                "end": 426
              }
            },
            "loc": {
              "start": 402,
              "end": 426
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 427,
                "end": 439
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 446,
                      "end": 448
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 446,
                    "end": 448
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 453,
                      "end": 461
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 453,
                    "end": 461
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 466,
                      "end": 469
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 466,
                    "end": 469
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 474,
                      "end": 478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 474,
                    "end": 478
                  }
                }
              ],
              "loc": {
                "start": 440,
                "end": 480
              }
            },
            "loc": {
              "start": 427,
              "end": 480
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 481,
                "end": 484
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 491,
                      "end": 504
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 491,
                    "end": 504
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 509,
                      "end": 518
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 509,
                    "end": 518
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 523,
                      "end": 534
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 523,
                    "end": 534
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 539,
                      "end": 548
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 539,
                    "end": 548
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 553,
                      "end": 562
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 553,
                    "end": 562
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 567,
                      "end": 574
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 567,
                    "end": 574
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 579,
                      "end": 591
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 579,
                    "end": 591
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 596,
                      "end": 604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 596,
                    "end": 604
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 609,
                      "end": 623
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 634,
                            "end": 636
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 634,
                          "end": 636
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 645,
                            "end": 655
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 645,
                          "end": 655
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 664,
                            "end": 674
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 664,
                          "end": 674
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 683,
                            "end": 690
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 683,
                          "end": 690
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 699,
                            "end": 710
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 699,
                          "end": 710
                        }
                      }
                    ],
                    "loc": {
                      "start": 624,
                      "end": 716
                    }
                  },
                  "loc": {
                    "start": 609,
                    "end": 716
                  }
                }
              ],
              "loc": {
                "start": 485,
                "end": 718
              }
            },
            "loc": {
              "start": 481,
              "end": 718
            }
          }
        ],
        "loc": {
          "start": 264,
          "end": 720
        }
      },
      "loc": {
        "start": 221,
        "end": 720
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 730,
          "end": 746
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 750,
            "end": 762
          }
        },
        "loc": {
          "start": 750,
          "end": 762
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 765,
                "end": 767
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 765,
              "end": 767
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 768,
                "end": 779
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 768,
              "end": 779
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 780,
                "end": 786
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 780,
              "end": 786
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 787,
                "end": 799
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 787,
              "end": 799
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 800,
                "end": 803
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 810,
                      "end": 823
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 810,
                    "end": 823
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 828,
                      "end": 837
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 828,
                    "end": 837
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 842,
                      "end": 853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 842,
                    "end": 853
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 858,
                      "end": 867
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 858,
                    "end": 867
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 872,
                      "end": 881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 872,
                    "end": 881
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 886,
                      "end": 893
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 886,
                    "end": 893
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 898,
                      "end": 910
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 898,
                    "end": 910
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 915,
                      "end": 923
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 915,
                    "end": 923
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 928,
                      "end": 942
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 953,
                            "end": 955
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 953,
                          "end": 955
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 964,
                            "end": 974
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 964,
                          "end": 974
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 983,
                            "end": 993
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 983,
                          "end": 993
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1002,
                            "end": 1009
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1002,
                          "end": 1009
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1018,
                            "end": 1029
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1018,
                          "end": 1029
                        }
                      }
                    ],
                    "loc": {
                      "start": 943,
                      "end": 1035
                    }
                  },
                  "loc": {
                    "start": 928,
                    "end": 1035
                  }
                }
              ],
              "loc": {
                "start": 804,
                "end": 1037
              }
            },
            "loc": {
              "start": 800,
              "end": 1037
            }
          }
        ],
        "loc": {
          "start": 763,
          "end": 1039
        }
      },
      "loc": {
        "start": 721,
        "end": 1039
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 1049,
          "end": 1061
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 1065,
            "end": 1072
          }
        },
        "loc": {
          "start": 1065,
          "end": 1072
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 1075,
                "end": 1083
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 1090,
                      "end": 1102
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1113,
                            "end": 1115
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1113,
                          "end": 1115
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1124,
                            "end": 1132
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1124,
                          "end": 1132
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1141,
                            "end": 1152
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1141,
                          "end": 1152
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1161,
                            "end": 1165
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1161,
                          "end": 1165
                        }
                      }
                    ],
                    "loc": {
                      "start": 1103,
                      "end": 1171
                    }
                  },
                  "loc": {
                    "start": 1090,
                    "end": 1171
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1176,
                      "end": 1178
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1176,
                    "end": 1178
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1183,
                      "end": 1193
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1183,
                    "end": 1193
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1198,
                      "end": 1208
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1198,
                    "end": 1208
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 1213,
                      "end": 1229
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1213,
                    "end": 1229
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1234,
                      "end": 1242
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1234,
                    "end": 1242
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1247,
                      "end": 1256
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1247,
                    "end": 1256
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1261,
                      "end": 1273
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1261,
                    "end": 1273
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 1278,
                      "end": 1294
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1278,
                    "end": 1294
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 1299,
                      "end": 1309
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1299,
                    "end": 1309
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1314,
                      "end": 1326
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1314,
                    "end": 1326
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1331,
                      "end": 1343
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1331,
                    "end": 1343
                  }
                }
              ],
              "loc": {
                "start": 1084,
                "end": 1345
              }
            },
            "loc": {
              "start": 1075,
              "end": 1345
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1346,
                "end": 1348
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1346,
              "end": 1348
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1349,
                "end": 1359
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1349,
              "end": 1359
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1360,
                "end": 1370
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1360,
              "end": 1370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1371,
                "end": 1380
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1371,
              "end": 1380
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1381,
                "end": 1392
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1381,
              "end": 1392
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1393,
                "end": 1399
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 1409,
                      "end": 1419
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1406,
                    "end": 1419
                  }
                }
              ],
              "loc": {
                "start": 1400,
                "end": 1421
              }
            },
            "loc": {
              "start": 1393,
              "end": 1421
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1422,
                "end": 1427
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1441,
                        "end": 1453
                      }
                    },
                    "loc": {
                      "start": 1441,
                      "end": 1453
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1467,
                            "end": 1483
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1464,
                          "end": 1483
                        }
                      }
                    ],
                    "loc": {
                      "start": 1454,
                      "end": 1489
                    }
                  },
                  "loc": {
                    "start": 1434,
                    "end": 1489
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1501,
                        "end": 1505
                      }
                    },
                    "loc": {
                      "start": 1501,
                      "end": 1505
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1519,
                            "end": 1527
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1516,
                          "end": 1527
                        }
                      }
                    ],
                    "loc": {
                      "start": 1506,
                      "end": 1533
                    }
                  },
                  "loc": {
                    "start": 1494,
                    "end": 1533
                  }
                }
              ],
              "loc": {
                "start": 1428,
                "end": 1535
              }
            },
            "loc": {
              "start": 1422,
              "end": 1535
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1536,
                "end": 1547
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1536,
              "end": 1547
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1548,
                "end": 1562
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1548,
              "end": 1562
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1563,
                "end": 1568
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1563,
              "end": 1568
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1569,
                "end": 1578
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1569,
              "end": 1578
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1579,
                "end": 1583
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 1593,
                      "end": 1601
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1590,
                    "end": 1601
                  }
                }
              ],
              "loc": {
                "start": 1584,
                "end": 1603
              }
            },
            "loc": {
              "start": 1579,
              "end": 1603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1604,
                "end": 1618
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1604,
              "end": 1618
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1619,
                "end": 1624
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1619,
              "end": 1624
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1625,
                "end": 1628
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1635,
                      "end": 1644
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1635,
                    "end": 1644
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1649,
                      "end": 1660
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1649,
                    "end": 1660
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1665,
                      "end": 1676
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1665,
                    "end": 1676
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1681,
                      "end": 1690
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1681,
                    "end": 1690
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1695,
                      "end": 1702
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1695,
                    "end": 1702
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1707,
                      "end": 1715
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1707,
                    "end": 1715
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1720,
                      "end": 1732
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1720,
                    "end": 1732
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1737,
                      "end": 1745
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1737,
                    "end": 1745
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1750,
                      "end": 1758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1750,
                    "end": 1758
                  }
                }
              ],
              "loc": {
                "start": 1629,
                "end": 1760
              }
            },
            "loc": {
              "start": 1625,
              "end": 1760
            }
          }
        ],
        "loc": {
          "start": 1073,
          "end": 1762
        }
      },
      "loc": {
        "start": 1040,
        "end": 1762
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 1772,
          "end": 1780
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 1784,
            "end": 1787
          }
        },
        "loc": {
          "start": 1784,
          "end": 1787
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1790,
                "end": 1792
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1790,
              "end": 1792
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1793,
                "end": 1803
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1793,
              "end": 1803
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 1804,
                "end": 1807
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1804,
              "end": 1807
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1808,
                "end": 1817
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1808,
              "end": 1817
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1818,
                "end": 1830
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1837,
                      "end": 1839
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1837,
                    "end": 1839
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1844,
                      "end": 1852
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1844,
                    "end": 1852
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1857,
                      "end": 1868
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1857,
                    "end": 1868
                  }
                }
              ],
              "loc": {
                "start": 1831,
                "end": 1870
              }
            },
            "loc": {
              "start": 1818,
              "end": 1870
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1871,
                "end": 1874
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 1881,
                      "end": 1886
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1881,
                    "end": 1886
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1891,
                      "end": 1903
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1891,
                    "end": 1903
                  }
                }
              ],
              "loc": {
                "start": 1875,
                "end": 1905
              }
            },
            "loc": {
              "start": 1871,
              "end": 1905
            }
          }
        ],
        "loc": {
          "start": 1788,
          "end": 1907
        }
      },
      "loc": {
        "start": 1763,
        "end": 1907
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 1917,
          "end": 1925
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 1929,
            "end": 1933
          }
        },
        "loc": {
          "start": 1929,
          "end": 1933
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1936,
                "end": 1938
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1936,
              "end": 1938
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1939,
                "end": 1949
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1939,
              "end": 1949
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1950,
                "end": 1960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1950,
              "end": 1960
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1961,
                "end": 1972
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1961,
              "end": 1972
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1973,
                "end": 1979
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1973,
              "end": 1979
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 1980,
                "end": 1985
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1980,
              "end": 1985
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBotDepictingPerson",
              "loc": {
                "start": 1986,
                "end": 2006
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1986,
              "end": 2006
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2007,
                "end": 2011
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2007,
              "end": 2011
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2012,
                "end": 2024
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2012,
              "end": 2024
            }
          }
        ],
        "loc": {
          "start": 1934,
          "end": 2026
        }
      },
      "loc": {
        "start": 1908,
        "end": 2026
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "projectOrOrganizations",
      "loc": {
        "start": 2034,
        "end": 2056
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2058,
              "end": 2063
            }
          },
          "loc": {
            "start": 2057,
            "end": 2063
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "ProjectOrOrganizationSearchInput",
              "loc": {
                "start": 2065,
                "end": 2097
              }
            },
            "loc": {
              "start": 2065,
              "end": 2097
            }
          },
          "loc": {
            "start": 2065,
            "end": 2098
          }
        },
        "directives": [],
        "loc": {
          "start": 2057,
          "end": 2098
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "projectOrOrganizations",
            "loc": {
              "start": 2104,
              "end": 2126
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2127,
                  "end": 2132
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2135,
                    "end": 2140
                  }
                },
                "loc": {
                  "start": 2134,
                  "end": 2140
                }
              },
              "loc": {
                "start": 2127,
                "end": 2140
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "edges",
                  "loc": {
                    "start": 2148,
                    "end": 2153
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "cursor",
                        "loc": {
                          "start": 2164,
                          "end": 2170
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2164,
                        "end": 2170
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 2179,
                          "end": 2183
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "selectionSet": {
                        "kind": "SelectionSet",
                        "selections": [
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Project",
                                "loc": {
                                  "start": 2205,
                                  "end": 2212
                                }
                              },
                              "loc": {
                                "start": 2205,
                                "end": 2212
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Project_list",
                                    "loc": {
                                      "start": 2234,
                                      "end": 2246
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2231,
                                    "end": 2246
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2213,
                                "end": 2260
                              }
                            },
                            "loc": {
                              "start": 2198,
                              "end": 2260
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Organization",
                                "loc": {
                                  "start": 2280,
                                  "end": 2292
                                }
                              },
                              "loc": {
                                "start": 2280,
                                "end": 2292
                              }
                            },
                            "directives": [],
                            "selectionSet": {
                              "kind": "SelectionSet",
                              "selections": [
                                {
                                  "kind": "FragmentSpread",
                                  "name": {
                                    "kind": "Name",
                                    "value": "Organization_list",
                                    "loc": {
                                      "start": 2314,
                                      "end": 2331
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 2311,
                                    "end": 2331
                                  }
                                }
                              ],
                              "loc": {
                                "start": 2293,
                                "end": 2345
                              }
                            },
                            "loc": {
                              "start": 2273,
                              "end": 2345
                            }
                          }
                        ],
                        "loc": {
                          "start": 2184,
                          "end": 2355
                        }
                      },
                      "loc": {
                        "start": 2179,
                        "end": 2355
                      }
                    }
                  ],
                  "loc": {
                    "start": 2154,
                    "end": 2361
                  }
                },
                "loc": {
                  "start": 2148,
                  "end": 2361
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 2366,
                    "end": 2374
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 2385,
                          "end": 2396
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2385,
                        "end": 2396
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 2405,
                          "end": 2421
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2405,
                        "end": 2421
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorOrganization",
                        "loc": {
                          "start": 2430,
                          "end": 2451
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 2430,
                        "end": 2451
                      }
                    }
                  ],
                  "loc": {
                    "start": 2375,
                    "end": 2457
                  }
                },
                "loc": {
                  "start": 2366,
                  "end": 2457
                }
              }
            ],
            "loc": {
              "start": 2142,
              "end": 2461
            }
          },
          "loc": {
            "start": 2104,
            "end": 2461
          }
        }
      ],
      "loc": {
        "start": 2100,
        "end": 2463
      }
    },
    "loc": {
      "start": 2028,
      "end": 2463
    }
  },
  "variableValues": {},
  "path": {
    "key": "projectOrOrganization_findMany"
  }
} as const;
